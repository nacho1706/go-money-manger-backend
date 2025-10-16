-- Update transaction type enum values from gasto/ingreso to expense/income
-- First, update the existing data
UPDATE `transactions` SET `type` = 'expense' WHERE `type` = 'gasto';
UPDATE `transactions` SET `type` = 'income' WHERE `type` = 'ingreso';

-- Then, alter the table to change the enum values
ALTER TABLE `transactions` MODIFY COLUMN `type` enum('expense','income') NOT NULL;