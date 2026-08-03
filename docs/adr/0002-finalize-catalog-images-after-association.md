# Finalize Catalog Images After Association

Catalog Images are uploaded as temporary objects and become owned only after their database association commits, with stale temporary objects cleaned asynchronously. PostgreSQL and object storage cannot share a transaction, and this approach avoids permanent upload leaks without relying on brittle synchronous compensation.
