fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Compile crypto-service proto for client
    tonic_build::configure()
        .build_server(false)
        .build_client(true)
        .compile_protos(
            &["proto/crypto_service.proto"],
            &["proto"],
        )?;

    // Compile auth-edge proto for server implementation
    // Using simplified version without buf/validate and google/api imports
    tonic_build::configure()
        .build_server(true)
        .build_client(false)
        .compile_protos(
            &["proto/auth_edge.proto"],
            &["proto"],
        )?;

    Ok(())
}
