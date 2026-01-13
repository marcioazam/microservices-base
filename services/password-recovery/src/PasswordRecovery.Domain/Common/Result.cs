namespace PasswordRecovery.Domain.Common;

public class Result
{
    public bool IsSuccess { get; }
    public bool IsFailure => !IsSuccess;
    public string? Error { get; }
    public IReadOnlyList<string> ValidationErrors { get; }

    protected Result(bool isSuccess, string? error, IReadOnlyList<string>? validationErrors = null)
    {
        IsSuccess = isSuccess;
        Error = error;
        ValidationErrors = validationErrors ?? [];
    }

    public static Result Success() => new(true, null);
    public static Result Failure(string error) => new(false, error);
    public static Result ValidationFailure(IEnumerable<string> errors) => new(false, "Validation failed", errors.ToList());
}

public class Result<T> : Result
{
    public T? Value { get; }

    private Result(bool isSuccess, T? value, string? error, IReadOnlyList<string>? validationErrors = null)
        : base(isSuccess, error, validationErrors)
    {
        Value = value;
    }

    public static Result<T> Success(T value) => new(true, value, null);
    public new static Result<T> Failure(string error) => new(false, default, error);
    public new static Result<T> ValidationFailure(IEnumerable<string> errors) => new(false, default, "Validation failed", errors.ToList());

    public TResult Match<TResult>(Func<T, TResult> onSuccess, Func<string, TResult> onFailure)
    {
        return IsSuccess ? onSuccess(Value!) : onFailure(Error!);
    }
}
