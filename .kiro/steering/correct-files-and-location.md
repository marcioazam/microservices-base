---
inclusion: always
---
# AGENT RULES - FILE SIZE & ARCHITECTURE ENFORCEMENT

**CRITICAL RULE**: Max 400 lines per file (non-blank) | Auto-validate EVERY create/edit | Auto-split if exceeded | No exceptions

## ARCHITECTURE STRUCTURE

src/
- api/controllers/middlewares/routes/
- services/{domain}/{name}.service|repository|validator.ts
- domain/entities/interfaces/
- infrastructure/database/cache/messaging/
- shared/utils/constants/types/
- config/

tests/ (MIRRORS src/)
- unit/services/{domain}/{name}.spec.ts
- integration/api|infrastructure/
- e2e/{feature}-flows/
- fixtures/mocks/

**Rules**: Test path = Source path | 1 source = 1+ test | Mirror structure mandatory

## AUTO-VALIDATION WORKFLOW

ON EVERY FILE OPERATION:
1. Count non-blank lines
2. IF >400 → HALT + analyze split points
3. Split by: multiple classes → 1/file | large class → extract helpers | mixed concerns → separate | long functions → extract
4. Update imports/exports
5. Move/split tests
6. Run test suite
7. Report changes

## SPLIT PATTERNS

**Class Extraction**: Split 450-line file into 3 files of ~150 lines each by extracting authentication, profile management, and notifications into separate services

**Helper Extraction**: Split 500-line service into core service (200 lines) plus separate calculator, validator, and payment processing helpers

**Feature Modules**: Split 600-line controller into separate user, product, and order controllers

**Test Organization**: Mirror source structure - src/services/user/user.service.ts creates tests/unit/services/user/user.service.spec.ts + tests/integration/services/user/user.integration.spec.ts + tests/e2e/user-flows/

## EXECUTION CHECKLIST

**Create**: Verify architecture placement | Check <400 lines | Create test file | Update exports

**Edit**: Count after edit | IF >400 → auto-split | Update imports | Re-run tests | Verify no regressions

**Split**: Extract logical units | Maintain SRP | Update imports | Move/split tests | Run suite | Document

## VIOLATIONS & AUTO-FIX

**Monolithic File**: 850-line file automatically split into 3 files (280+290+280) with import updates and test verification

**Wrong Test Location**: Move from tests/payment.spec.ts to tests/unit/services/payment/payment.service.spec.ts with import updates

**Mixed Test Types**: Split 500-line mixed test into separate unit, integration, and e2e test files

## AUTOMATION TRIGGERS

**"create service"**:
1. Create src/services/{domain}/{domain}.service|repository|validator.ts
2. Create tests/unit/services/{domain}/{domain}.service.spec.ts
3. Create tests/integration/services/{domain}/{domain}.integration.spec.ts
4. Add barrel exports
5. Validate <400 lines each

**File >400 lines**:
1. Analyze structure (classes/functions/concerns)
2. Identify boundaries
3. Propose split strategy
4. Execute: extract → update imports → move tests → run suite
5. Report with file tree

## CI/CD GATES

pre-commit:
- Check all TypeScript files for >400 lines → FAIL
- Verify src/ tree matches tests/unit/ tree → FAIL if not mirrored
- Run jest coverage → FAIL if <80% per module

build:
- Max lines: 400
- Test ratio: 1:1
- Architecture compliance: 100%

## COMMANDS

npm run lint:file-size - Check violations
npm run test:validate-structure - Verify mirror
npm run test:fix-structure - Auto-fix
npm run test:generate-missing - Create missing tests

## CORE PRINCIPLES

400 lines = HARD limit | Auto-split mandatory | Test parity required | Mirror structure enforced | SRP always | No exceptions | CI enforced | Proactive validation

**SUCCESS**: Every file <400 lines | Test mirrors source | Zero violations | 100% CI compliance