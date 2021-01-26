create database archondb;
CREATE USER archonadmin WITH ENCRYPTED PASSWORD 'psoadminpassword';
GRANT ALL ON ALL TABLES IN SCHEMA public TO archonadmin;
