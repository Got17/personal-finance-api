package messages

const (
	MsgTransferNotFound                  = "transfer not found"
	MsgTransferAccessDenied              = "access denied to transfer"
	MsgTransferAccountsMustBeDistinct    = "source and destination accounts must be distinct"
	MsgTransferSameCurrencyMismatch      = "same-currency transfer amounts must be equal"
	MsgTransferDestinationAmountMismatch = "destination amount does not match rate conversion precision"
	MsgTransferRateMustBePositive        = "exchange rate must be greater than zero"
	MsgFXQuoteNotFound                   = "fx quote not found"
	MsgFXRateUnavailable                 = "exchange rate is unavailable for the requested currency pair and date"
	MsgInsufficientAccountBalance        = "insufficient account balance"
	MsgInsufficientFeeAccountBalance     = "insufficient fee account balance"
)

