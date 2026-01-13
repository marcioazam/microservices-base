ExUnit.start()

# Configure ExUnitProperties for property-based testing
Application.put_env(:stream_data, :max_runs, 100)
