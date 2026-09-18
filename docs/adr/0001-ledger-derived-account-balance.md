# Ledger-Derived Account Balance and Non-Negative Transfer Enforcement

## Status
Accepted

## Context
When executing transfers between accounts, the system must ensure the source account (and any fee-paying account) holds sufficient funds. Accounts begin with a zero balance and are funded by income records or inbound transfers. Without balance validation and concurrency controls, accounts could enter negative balances or suffer from race conditions (double spending). Additionally, supporting both asset holdings and liability/credit accounts with differing overdraft semantics adds unnecessary complexity.

## Decision
1. **Dynamic Ledger-Derived Balance**: Compute an account's available balance on-the-fly by summing active inflows (income records, inbound transfer legs) minus active outflows (expense records, outbound transfer legs, transfer fees) from `financial_records`. We do not persist a mutable `balance_minor` column on the `accounts` table.
2. **Pessimistic Locking**: During transfer creation, wrap the balance check and transfer insertion in a database transaction that acquires a row-level lock on the source account (and distinct fee account if applicable) using `SELECT ... FOR UPDATE`.
3. **Asset Holdings Only**: Remove liability/credit accounts (`credit_card`, `loan`) from the domain model and supported account types, treating all accounts as depository/asset holdings that strictly enforce non-negative balances.
4. **Validation Contract**: Return HTTP 422 Unprocessable Entity with targeted field errors (`source_account_id: "insufficient account balance"`, `fee.account_id: "insufficient account balance"`) if available balance is less than the required outlay.

## Considered Options
- **Materialized Balance Column on Accounts**: Storing a running balance on `accounts` requires dual-writes and continuous synchronization across income, expense, transfer, archive, and update operations, introducing risks of ledger drift. Dynamic aggregation off immutable financial records guarantees a single source of truth.
- **Optimistic Concurrency Control**: Fails fast under low contention but would require retry loops on concurrent transfers. Pessimistic row locking on the account cleanly serializes debit operations.
- **Credit Limit & Overdraft Support**: Allowing negative balances for credit card or loan accounts complicates transfer preconditions. Removing liability accounts simplifies domain rules to uniform non-negative asset balances.

## Consequences
- Every account balance is strictly mathematically provable from historical active records.
- Outbound transfers are blocked when balance is zero or insufficient, satisfying the requirement that accounts must be funded before transferring out.
- The `accounts` schema and API validation are streamlined to asset account types (`checking`, `savings`, `investment`, `cash`, `other`).
