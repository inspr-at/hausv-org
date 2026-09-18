-- Product role: NOSUPERUSER NOBYPASSRLS (the application refuses anything else).
CREATE ROLE hausv LOGIN PASSWORD 'hausv-demo' NOSUPERUSER NOBYPASSRLS;
ALTER DATABASE hausv OWNER TO hausv;
GRANT ALL ON SCHEMA public TO hausv;
