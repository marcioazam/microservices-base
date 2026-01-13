// Package controller implements the ResiliencePolicy reconciliation logic.
package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	resiliencev1 "github.com/auth-platform/platform/resilience-operator/api/v1"
)

const (
	finalizerName = "resilience.auth-platform.github.com/finalizer"

	// Condition types
	ConditionTypeReady = "Ready"

	// Condition reasons
	ReasonApplied        = "Applied"
	ReasonTargetNotFound = "TargetServiceNotFound"
	ReasonReconciling    = "Reconciling"
	ReasonFailed         = "Failed"
)

// ResiliencePolicyReconciler reconciles a ResiliencePolicy object.
type ResiliencePolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=resilience.auth-platform.github.com,resources=resiliencepolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=resilience.auth-platform.github.com,resources=resiliencepolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=resilience.auth-platform.github.com,resources=resiliencepolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=grpcroutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;update;patch

// Reconcile handles ResiliencePolicy reconciliation.
func (r *ResiliencePolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconciling ResiliencePolicy", "name", req.Name, "namespace", req.Namespace)

	policy := &resiliencev1.ResiliencePolicy{}
	if err := r.Get(ctx, req.NamespacedName, policy); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("ResiliencePolicy not found, ignoring")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !policy.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, policy)
	}

	if !controllerutil.ContainsFinalizer(policy, finalizerName) {
		controllerutil.AddFinalizer(policy, finalizerName)
		if err := r.Update(ctx, policy); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	targetService, err := r.getTargetService(ctx, policy)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("target service not found", "service", policy.Spec.TargetRef.Name)
			r.setStatusCondition(ctx, policy, metav1.Condition{
				Type:    ConditionTypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  ReasonTargetNotFound,
				Message: fmt.Sprintf("Target service %s not found", policy.Spec.TargetRef.Name),
			})
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.applyCircuitBreaker(ctx, policy, targetService); err != nil {
		logger.Error(err, "failed to apply circuit breaker")
		r.setStatusCondition(ctx, policy, metav1.Condition{
			Type:    ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonFailed,
			Message: fmt.Sprintf("Failed to apply circuit breaker: %v", err),
		})
		return ctrl.Result{}, err
	}

	if err := r.applyRetryAndTimeout(ctx, policy, targetService); err != nil {
		logger.Error(err, "failed to apply retry and timeout")
		r.setStatusCondition(ctx, policy, metav1.Condition{
			Type:    ConditionTypeReady,
			Status:  metav1.ConditionFalse,
			Reason:  ReasonFailed,
			Message: fmt.Sprintf("Failed to apply retry/timeout: %v", err),
		})
		return ctrl.Result{}, err
	}

	r.setStatusCondition(ctx, policy, metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonApplied,
		Message: "Resilience policy successfully applied",
	})

	logger.Info("successfully reconciled ResiliencePolicy", "name", req.Name)
	return ctrl.Result{}, nil
}


// getTargetService retrieves the target service for the policy.
func (r *ResiliencePolicyReconciler) getTargetService(ctx context.Context, policy *resiliencev1.ResiliencePolicy) (*corev1.Service, error) {
	targetNamespace := policy.Spec.TargetRef.Namespace
	if targetNamespace == "" {
		targetNamespace = policy.Namespace
	}

	service := &corev1.Service{}
	key := types.NamespacedName{
		Name:      policy.Spec.TargetRef.Name,
		Namespace: targetNamespace,
	}

	if err := r.Get(ctx, key, service); err != nil {
		return nil, err
	}

	return service, nil
}

// applyCircuitBreaker configures circuit breaking via Service annotations.
func (r *ResiliencePolicyReconciler) applyCircuitBreaker(ctx context.Context, policy *resiliencev1.ResiliencePolicy, service *corev1.Service) error {
	if service.Annotations == nil {
		service.Annotations = make(map[string]string)
	}

	if policy.Spec.CircuitBreaker == nil || !policy.Spec.CircuitBreaker.Enabled {
		delete(service.Annotations, "config.linkerd.io/failure-accrual")
		delete(service.Annotations, "config.linkerd.io/failure-accrual-consecutive-failures")
	} else {
		service.Annotations["config.linkerd.io/failure-accrual"] = "consecutive"
		service.Annotations["config.linkerd.io/failure-accrual-consecutive-failures"] = fmt.Sprintf("%d", policy.Spec.CircuitBreaker.FailureThreshold)
	}

	return r.Update(ctx, service)
}

// applyRetryAndTimeout configures retries and timeouts via HTTPRoute.
func (r *ResiliencePolicyReconciler) applyRetryAndTimeout(ctx context.Context, policy *resiliencev1.ResiliencePolicy, service *corev1.Service) error {
	if (policy.Spec.Retry == nil || !policy.Spec.Retry.Enabled) &&
		(policy.Spec.Timeout == nil || !policy.Spec.Timeout.Enabled) {
		return r.deleteHTTPRoute(ctx, policy, service)
	}

	httpRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-resilience", service.Name),
			Namespace: service.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, httpRoute, func() error {
		if err := controllerutil.SetControllerReference(policy, httpRoute, r.Scheme); err != nil {
			return err
		}

		annotations := make(map[string]string)

		if policy.Spec.Retry != nil && policy.Spec.Retry.Enabled {
			annotations["retry.linkerd.io/http"] = fmt.Sprintf("%d", policy.Spec.Retry.MaxAttempts)
			if policy.Spec.Retry.RetryableStatusCodes != "" {
				annotations["retry.linkerd.io/http-status-codes"] = policy.Spec.Retry.RetryableStatusCodes
			}
			if policy.Spec.Retry.RetryTimeout != "" {
				annotations["retry.linkerd.io/timeout"] = policy.Spec.Retry.RetryTimeout
			}
		}

		if policy.Spec.Timeout != nil && policy.Spec.Timeout.Enabled {
			annotations["timeout.linkerd.io/request"] = policy.Spec.Timeout.RequestTimeout
			if policy.Spec.Timeout.ResponseTimeout != "" {
				annotations["timeout.linkerd.io/response"] = policy.Spec.Timeout.ResponseTimeout
			}
		}

		httpRoute.Annotations = annotations

		port := gatewayv1.PortNumber(80)
		if policy.Spec.TargetRef.Port != nil {
			port = gatewayv1.PortNumber(*policy.Spec.TargetRef.Port)
		}

		kind := gatewayv1.Kind("Service")
		httpRoute.Spec = gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name: gatewayv1.ObjectName(service.Name),
						Kind: &kind,
					},
				},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: gatewayv1.ObjectName(service.Name),
									Port: &port,
								},
							},
						},
					},
				},
			},
		}

		return nil
	})

	return err
}


// deleteHTTPRoute removes the HTTPRoute if it exists.
func (r *ResiliencePolicyReconciler) deleteHTTPRoute(ctx context.Context, policy *resiliencev1.ResiliencePolicy, service *corev1.Service) error {
	httpRoute := &gatewayv1.HTTPRoute{}
	key := types.NamespacedName{
		Name:      fmt.Sprintf("%s-resilience", service.Name),
		Namespace: service.Namespace,
	}

	if err := r.Get(ctx, key, httpRoute); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return r.Delete(ctx, httpRoute)
}

// handleDeletion removes all applied configurations and removes finalizer.
func (r *ResiliencePolicyReconciler) handleDeletion(ctx context.Context, policy *resiliencev1.ResiliencePolicy) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("handling deletion of ResiliencePolicy", "name", policy.Name)

	targetService, err := r.getTargetService(ctx, policy)
	if err == nil {
		delete(targetService.Annotations, "config.linkerd.io/failure-accrual")
		delete(targetService.Annotations, "config.linkerd.io/failure-accrual-consecutive-failures")
		if err := r.Update(ctx, targetService); err != nil {
			logger.Error(err, "failed to clean up service annotations")
		}

		r.deleteHTTPRoute(ctx, policy, targetService)
	}

	controllerutil.RemoveFinalizer(policy, finalizerName)
	if err := r.Update(ctx, policy); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("successfully deleted ResiliencePolicy", "name", policy.Name)
	return ctrl.Result{}, nil
}

// setStatusCondition updates the ResiliencePolicy status.
func (r *ResiliencePolicyReconciler) setStatusCondition(ctx context.Context, policy *resiliencev1.ResiliencePolicy, condition metav1.Condition) {
	condition.LastTransitionTime = metav1.Now()
	condition.ObservedGeneration = policy.Generation

	found := false
	for i := range policy.Status.Conditions {
		if policy.Status.Conditions[i].Type == condition.Type {
			policy.Status.Conditions[i] = condition
			found = true
			break
		}
	}
	if !found {
		policy.Status.Conditions = append(policy.Status.Conditions, condition)
	}

	policy.Status.ObservedGeneration = policy.Generation
	now := metav1.Now()
	policy.Status.LastUpdateTime = &now

	targetNamespace := policy.Spec.TargetRef.Namespace
	if targetNamespace == "" {
		targetNamespace = policy.Namespace
	}
	policy.Status.AppliedToServices = []string{
		fmt.Sprintf("%s/%s", targetNamespace, policy.Spec.TargetRef.Name),
	}

	if err := r.Status().Update(ctx, policy); err != nil {
		log.FromContext(ctx).Error(err, "failed to update status")
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResiliencePolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&resiliencev1.ResiliencePolicy{}).
		Owns(&gatewayv1.HTTPRoute{}).
		Complete(r)
}
