-- Role-Based Access Control (RBAC) Initialization for PostgreSQL

-- 1. Create Roles with specified credentials safely
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'sani') THEN
        CREATE ROLE sani WITH LOGIN PASSWORD 'Sani123';
    END IF;
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'elec') THEN
        CREATE ROLE elec WITH LOGIN PASSWORD 'Elec123';
    END IF;
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'poss') THEN
        CREATE ROLE poss WITH LOGIN PASSWORD 'Poss123';
    END IF;
END
$$;

-- 2. Create dedicated schemas for isolation
-- This ensures they "only access their own creations" by default
CREATE SCHEMA sani AUTHORIZATION sani;
CREATE SCHEMA elec AUTHORIZATION elec;
CREATE SCHEMA poss AUTHORIZATION poss;

-- 3. Set default search paths
-- Users will look into their own schema first, then public for shared tables
ALTER ROLE sani SET search_path TO sani, public;
ALTER ROLE elec SET search_path TO elec, public;
ALTER ROLE poss SET search_path TO poss, public;

-- 4. Secure public schema
-- Prevent regular users from creating tables in the public schema
REVOKE CREATE ON SCHEMA public FROM PUBLIC;

-- 5. Special permissions for 'Poss'
-- Poss needs to see 'suhu' (temp) and 'mdcw' (production_mdcw) in the public schema
GRANT USAGE ON SCHEMA public TO poss;

-- Grant SELECT only on the requested shared tables
-- We use DO block to handle cases where tables might not exist yet during init
DO $$ 
BEGIN 
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'temp') THEN
        GRANT SELECT ON public.temp TO poss;
    END IF;
    
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'production_mdcw') THEN
        GRANT SELECT ON public.production_mdcw TO poss;
    END IF;

    -- Also grant select on 'ck' if mdcw refers to that
    IF EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'ck') THEN
        GRANT SELECT ON public.ck TO poss;
    END IF;
END $$;

-- 6. Ensure full privileges on their own schemas
GRANT ALL PRIVILEGES ON SCHEMA sani TO sani;
GRANT ALL PRIVILEGES ON SCHEMA elec TO elec;
GRANT ALL PRIVILEGES ON SCHEMA poss TO poss;

-- 7. Default Privileges
-- Ensures that any new tables created in public by 'postgres' are visible to 'poss'
-- (if they are the specific ones requested)
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO poss;
