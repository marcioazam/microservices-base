-- Create separate databases for each microservice
-- Following the pattern: one database per service, same PostgreSQL instance

-- Auth Platform databases
CREATE DATABASE auth_db;
CREATE DATABASE session_db;
CREATE DATABASE mfa_db;
CREATE DATABASE iam_db;

-- Platform services databases
CREATE DATABASE logging_db;
CREATE DATABASE resilience_db;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE auth_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE session_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE mfa_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE iam_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE logging_db TO postgres;
GRANT ALL PRIVILEGES ON DATABASE resilience_db TO postgres;
