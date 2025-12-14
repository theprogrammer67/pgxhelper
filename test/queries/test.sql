--SQL:CreateTableCustomers
CREATE TABLE customers
(
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE
);
--end

--SQL:InsertCustomer
INSERT INTO customers (id, name, email) VALUES ($1, $2, $3);
--end

--SQL:GetCustomer
SELECT * FROM customers WHERE id = $1;
--end

--SQL:GetCustomers
SELECT * 
FROM customers 
WHERE id = ANY($1::text[]);
--end