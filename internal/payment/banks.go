package payment

// BankOption is one VietQR-supported bank for SePay top-ups.
type BankOption struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// SupportedBanks lists popular Vietnamese banks for VietQR image URLs.
var SupportedBanks = []BankOption{
	{Code: "MB", Name: "MB Bank"},
	{Code: "VCB", Name: "Vietcombank"},
	{Code: "TCB", Name: "Techcombank"},
	{Code: "ICB", Name: "VietinBank"},
	{Code: "BIDV", Name: "BIDV"},
	{Code: "ACB", Name: "ACB"},
	{Code: "VPB", Name: "VPBank"},
	{Code: "TPB", Name: "TPBank"},
	{Code: "STB", Name: "Sacombank"},
	{Code: "VIB", Name: "VIB"},
	{Code: "MSB", Name: "MSB"},
	{Code: "SHB", Name: "SHB"},
	{Code: "HDB", Name: "HDBank"},
	{Code: "VBA", Name: "Agribank"},
}