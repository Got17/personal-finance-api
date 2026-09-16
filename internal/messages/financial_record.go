package messages

const (
	MsgFinancialRecordNotFound        = "financial record not found"
	MsgFinancialRecordAccessDenied    = "access denied to financial record"
	MsgUnsupportedFinancialRecordKind = "unsupported financial record kind"
	MsgAccountMustBeActive            = "account must be active"
	MsgCategoryMustBeActive           = "category must be active"
	MsgCategoryKindMismatch           = "category type must match financial record kind"
	MsgAccountCurrencyMismatch        = "currency must match account currency"
	MsgDateIsRequired                 = "date is required"
	MsgInvalidDateRange               = "start date must not be after end date"
	MsgFinancialRecordArchived        = "archived financial record cannot be changed"
	MsgAmountMustBePositive           = "amount must be greater than zero"
)
