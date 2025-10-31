$body = @{
  order_id = "ORD123456"
  account_id = "ACC987654"
  account_iban = "1234"
  account_bic = "DEUTDEFF"
  checksum = "abc123xyz"
  transfers = @(
    @{
      merchant_trx_id = "TRX001"
      amount = "100.50"
      beneficiary_name = "John Doe"
      beneficiary_iban = "GB29NWBK60161331926819"
      beneficiary_bic = "NWBKGB2L"
    },
    @{
      merchant_trx_id = "TRX002"
      amount = "200.75"
      beneficiary_name = "Jane Smith"
      beneficiary_iban = "FR1420041010050500013M02606"
      beneficiary_bic = "CRLYFRPP"
    }
  )
} | ConvertTo-Json -Depth 10

Invoke-RestMethod -Uri "http://localhost:8080/transfer/bulk" -Method Post -ContentType "application/json" -Body $body