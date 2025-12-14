# Import Fix Progress

## Status: IN PROGRESS

### Completed ✅
1. Updated `magda-api/go.mod` to add dependencies on `magda-agents` and `magda-dsl`
2. Added replace directives for local development
3. Moved `magda-agents/go.mod` to `magda-agents/go/go.mod`
4. Fixed handler file imports (generation.go, magda.go, auth.go, etc.)
5. Started fixing agent file imports

### Current Issues ❌

1. **Internal Package Restriction**: Go prevents importing `internal/` packages from external modules
   - Solution: Need to restructure `magda-agents` to not use `internal/` directory OR make packages public

2. **Old Import Paths**: `magda-agents` files still have old import paths
   - Solution: Already fixed with sed command

3. **Package Name Mismatch**: `dsl_parser_test.go` has wrong package name
   - Solution: Already fixed (changed to `package daw`)

### Next Steps

1. **Restructure magda-agents**: Remove `internal/` directory structure OR make it importable
2. **Fix all import paths**: Ensure consistent module paths
3. **Test compilation**: Build and verify everything works

### Architecture Decision Needed

**Option A:** Remove `internal/` from `magda-agents`
- Make packages public: `github.com/Conceptual-Machines/magda-agents/go/agents/arranger`
- Simpler imports
- Can be imported by external modules

**Option B:** Keep `internal/` but only for intra-module use
- Packages can't be imported from `magda-api`
- Would need to duplicate or restructure

**Recommendation:** Option A - make packages public since they're meant to be imported










