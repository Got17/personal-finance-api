package messages

const (
	MsgTransferNotFound                  = "transfer not found"
	MsgTransferAccessDenied              = "access denied to transfer"
	MsgTransferAccountsMustBeDistinct    = "source and destination accounts must be distinct"
	MsgTransferCannotHaveCategory        = "transfers must not have a category"
	MsgTransferSameCurrencyMismatch      = "same-currency transfer amounts must be equal"
	MsgTransferDestinationAmountMismatch = "destination amount does not match rate conversion precision"
	MsgTransferRateRequired              = "exchange rate is required for cross-currency transfers"
	MsgTransferRateMustBePositive        = "exchange rate must be greater than zero"
	MsgTransferSourceAmountPositive      = "source amount must be greater than zero"
	MsgTransferDestAmountPositive        = "destination amount must be greater than zero"
	MsgFeeAccountMustBeActive            = "fee account must be active"
	MsgFeeCategoryMustBeExpense          = "fee category must be an active expense category"
	MsgFeeCurrencyMismatch               = "fee currency must match fee account currency"
	MsgFeeAmountMustBePositive           = "fee amount must be greater than zero"
	MsgFXQuoteNotFound                   = "fx quote not found"
	MsgFXRateUnavailable                 = "exchange rate is unavailable for the requested currency pair and date"
	MsgTransferCreated                   = "transfer created successfully"
)
