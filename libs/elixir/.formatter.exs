# Used by "mix format"
[
  inputs: ["{mix,.formatter}.exs", "{config,lib,test}/**/*.{ex,exs}"],
  subdirectories: ["apps/*"],
  line_length: 100,
  import_deps: [:stream_data],
  locals_without_parens: [
    # ExUnit
    assert: 1,
    assert: 2,
    refute: 1,
    refute: 2,
    # StreamData
    check: 1,
    check: 2
  ]
]
