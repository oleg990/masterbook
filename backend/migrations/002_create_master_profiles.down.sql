DROP TABLE master_profiles;


docker exec masterbook-postgres-dev `psql -U masterbook -d masterbook `-c "\dt"