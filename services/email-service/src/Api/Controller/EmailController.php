<?php

declare(strict_types=1);

namespace EmailService\Api\Controller;

use EmailService\Api\Request\AuditQueryRequest;
use EmailService\Api\Request\ResendEmailRequest;
use EmailService\Api\Request\SendEmailRequest;
use EmailService\Api\Response\ErrorResponse;
use EmailService\Application\DTO\EmailDTO;
use EmailService\Application\Service\AuditServiceInterface;
use EmailService\Application\Service\EmailServiceInterface;
use Symfony\Component\HttpFoundation\JsonResponse;
use Symfony\Component\HttpFoundation\Request;
use Symfony\Component\HttpFoundation\Response;
use Symfony\Component\Uid\Uuid;

class EmailController
{
    public function __construct(
        private readonly EmailServiceInterface $emailService,
        private readonly ?AuditServiceInterface $auditService = null
    ) {
    }

    public function send(Request $request): JsonResponse
    {
        $correlationId = $this->getCorrelationId($request);
        
        try {
            $data = json_decode($request->getContent(), true) ?? [];
            $sendRequest = SendEmailRequest::fromArray($data);
            
            $errors = $sendRequest->validate();
            if (!empty($errors)) {
                return $this->errorResponse(
                    ErrorResponse::validationError($errors, $correlationId),
                    Response::HTTP_BAD_REQUEST
                );
            }

            $dto = new EmailDTO(
                from: $sendRequest->from,
                recipients: $sendRequest->recipients,
                subject: $sendRequest->subject,
                body: $sendRequest->body,
                contentType: $sendRequest->getContentType(),
                cc: $sendRequest->cc,
                bcc: $sendRequest->bcc,
                attachments: $sendRequest->attachments,
                metadata: $sendRequest->metadata,
                type: $sendRequest->getEmailType(),
                templateId: $sendRequest->templateId,
                templateVariables: $sendRequest->templateVariables,
                fromName: $sendRequest->fromName
            );

            $result = $this->emailService->send($dto);

            if (!$result->success) {
                return $this->jsonResponse([
                    'success' => false,
                    'error' => [
                        'code' => $result->errorCode,
                        'message' => $result->errorMessage,
                    ],
                    'correlation_id' => $correlationId,
                ], Response::HTTP_BAD_REQUEST);
            }

            return $this->jsonResponse([
                'success' => true,
                'data' => [
                    'email_id' => $result->emailId,
                    'message_id' => $result->messageId,
                    'status' => $result->status?->value,
                ],
                'correlation_id' => $correlationId,
            ], Response::HTTP_OK);
        } catch (\Exception $e) {
            return $this->errorResponse(
                ErrorResponse::internalError($correlationId),
                Response::HTTP_INTERNAL_SERVER_ERROR
            );
        }
    }

    public function verify(Request $request): JsonResponse
    {
        $correlationId = $this->getCorrelationId($request);
        
        try {
            $data = json_decode($request->getContent(), true) ?? [];
            
            // Verification emails are a special type
            $data['type'] = 'verification';
            
            $sendRequest = SendEmailRequest::fromArray($data);
            
            $errors = $sendRequest->validate();
            if (!empty($errors)) {
                return $this->errorResponse(
                    ErrorResponse::validationError($errors, $correlationId),
                    Response::HTTP_BAD_REQUEST
                );
            }

            $dto = new EmailDTO(
                from: $sendRequest->from,
                recipients: $sendRequest->recipients,
                subject: $sendRequest->subject,
                body: $sendRequest->body,
                contentType: $sendRequest->getContentType(),
                type: $sendRequest->getEmailType(),
                templateId: $sendRequest->templateId,
                templateVariables: $sendRequest->templateVariables
            );

            $emailId = $this->emailService->sendAsync($dto);

            return $this->jsonResponse([
                'success' => true,
                'data' => [
                    'email_id' => $emailId,
                    'status' => 'queued',
                ],
                'correlation_id' => $correlationId,
            ], Response::HTTP_ACCEPTED);
        } catch (\Exception $e) {
            return $this->errorResponse(
                ErrorResponse::internalError($correlationId),
                Response::HTTP_INTERNAL_SERVER_ERROR
            );
        }
    }

    public function resend(Request $request): JsonResponse
    {
        $correlationId = $this->getCorrelationId($request);
        
        try {
            $data = json_decode($request->getContent(), true) ?? [];
            $resendRequest = ResendEmailRequest::fromArray($data);
            
            $errors = $resendRequest->validate();
            if (!empty($errors)) {
                return $this->errorResponse(
                    ErrorResponse::validationError($errors, $correlationId),
                    Response::HTTP_BAD_REQUEST
                );
            }

            $result = $this->emailService->resend($resendRequest->emailId);

            if (!$result->success) {
                $statusCode = $result->errorCode === 'EMAIL_NOT_FOUND' 
                    ? Response::HTTP_NOT_FOUND 
                    : Response::HTTP_BAD_REQUEST;
                    
                return $this->jsonResponse([
                    'success' => false,
                    'error' => [
                        'code' => $result->errorCode,
                        'message' => $result->errorMessage,
                    ],
                    'correlation_id' => $correlationId,
                ], $statusCode);
            }

            return $this->jsonResponse([
                'success' => true,
                'data' => [
                    'email_id' => $result->emailId,
                    'status' => $result->status?->value,
                ],
                'correlation_id' => $correlationId,
            ], Response::HTTP_OK);
        } catch (\Exception $e) {
            return $this->errorResponse(
                ErrorResponse::internalError($correlationId),
                Response::HTTP_INTERNAL_SERVER_ERROR
            );
        }
    }

    public function getStatus(Request $request, string $emailId): JsonResponse
    {
        $correlationId = $this->getCorrelationId($request);
        
        try {
            $status = $this->emailService->getStatus($emailId);

            return $this->jsonResponse([
                'success' => true,
                'data' => [
                    'email_id' => $emailId,
                    'status' => $status->value,
                ],
                'correlation_id' => $correlationId,
            ], Response::HTTP_OK);
        } catch (\RuntimeException $e) {
            return $this->errorResponse(
                ErrorResponse::notFound('Email', $correlationId),
                Response::HTTP_NOT_FOUND
            );
        } catch (\Exception $e) {
            return $this->errorResponse(
                ErrorResponse::internalError($correlationId),
                Response::HTTP_INTERNAL_SERVER_ERROR
            );
        }
    }

    public function getAuditLogs(Request $request): JsonResponse
    {
        $correlationId = $this->getCorrelationId($request);
        
        if ($this->auditService === null) {
            return $this->errorResponse(
                new ErrorResponse('SERVICE_UNAVAILABLE', 'Audit service not available', correlationId: $correlationId),
                Response::HTTP_SERVICE_UNAVAILABLE
            );
        }

        try {
            $queryParams = $request->query->all();
            $auditRequest = AuditQueryRequest::fromArray($queryParams);
            
            $errors = $auditRequest->validate();
            if (!empty($errors)) {
                return $this->errorResponse(
                    ErrorResponse::validationError($errors, $correlationId),
                    Response::HTTP_BAD_REQUEST
                );
            }

            $result = $this->auditService->query($auditRequest->toAuditQuery());

            return $this->jsonResponse([
                'success' => true,
                'data' => [
                    'items' => array_map(fn($log) => $log->toArray(), $result->items),
                    'total' => $result->total,
                    'limit' => $result->limit,
                    'offset' => $result->offset,
                    'has_more' => $result->hasMore(),
                ],
                'correlation_id' => $correlationId,
            ], Response::HTTP_OK);
        } catch (\Exception $e) {
            return $this->errorResponse(
                ErrorResponse::internalError($correlationId),
                Response::HTTP_INTERNAL_SERVER_ERROR
            );
        }
    }

    private function getCorrelationId(Request $request): string
    {
        return $request->headers->get('X-Correlation-ID', Uuid::v4()->toRfc4122());
    }

    private function jsonResponse(array $data, int $status): JsonResponse
    {
        return new JsonResponse($data, $status);
    }

    private function errorResponse(ErrorResponse $error, int $status): JsonResponse
    {
        return new JsonResponse($error->toArray(), $status);
    }
}
