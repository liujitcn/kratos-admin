# RPC

This directory contains TypeScript generated from the current module's Proto dependencies. Preserve the full Proto path, for example:

```text
src/rpc
├── base/v1
├── common/v1
└── system/admin/v1
```

Generate only the services and transitive types used by this module's APIs and pages. Generated files belong to this module and are exposed through the package's `./rpc/*` subpath. Do not generate business RPCs into core or duplicate their models by hand.

Update RPC files only with the generation command configured by the consuming project. Keep relative imports exactly as emitted; do not flatten the directory structure for appearance.
