# Personal Finance Hub

The core domain for personal financial tracking, multi-currency budgeting, and ledger management.

## Language

**Account**:
A financial holding or liability container owned by a user with an ISO 4217 currency.
_Avoid_: Bank, wallet, fund

**Category**:
A classification for grouping income or expense cash flows.
_Avoid_: Tag, label, bucket

**Financial Record**:
An entry recording an income or expense cash flow tied to an Account and a Category.
_Avoid_: Transaction, ledger item, entry

**Active Reference**:
An Account or Category that is owned by the authenticated user and currently active (`is_active = true`), permitting new or updated Financial Records to link to it.
_Avoid_: Valid foreign key, live target, valid reference
