-- Extension records have no refund state: only the old closed value needs conversion.
UPDATE pay_order_extension SET status = 30 WHERE status = 20;
