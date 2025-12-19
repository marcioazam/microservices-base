---
inclusion: always
---
# ARCHITECTURAL AGENT - ENFORCEMENT PROTOCOL

## FILE SIZE CONSTRAINT
**HARD LIMIT**: 400 non-blank lines per file
**VALIDATION**: Execute on every file operation
**ACTION**: Auto-split when threshold exceeded
**ENFORCEMENT**: Zero tolerance policy

## DIRECTORY STRUCTURE

```
src/
├── api/
│   ├── controllers/
│   ├── middlewares/
│   └── routes/
├── services/
│   └── {domain}/
│       ├── {name}.service.ts
│       ├── {name}.repository.ts
│       └── {name}.validator.ts
├── domain/
│   ├── entities/
│   └── interfaces/
├── infrastructure/
│   ├── database/
│   ├── cache/
│   └── messaging/
├── shared/
│   ├── utils/
│   ├── constants/
│   └── types/
└── config/

tests/
├── unit/
│   └── services/{domain}/{name}.spec.ts
├── integration/
│   └── api|infrastructure/
├── e2e/
│   └── {feature}-flows/
├── fixtures/
└── mocks/
```

## OPERATIONAL PROTOCOL

### Pre-Operation Validation
1. Calculate non-blank line count
2. Verify architectural placement
3. Validate naming conventions
4. Check dependency graph

### Threshold Breach Response
1. Halt operation immediately
2. Analyze structural boundaries
3. Identify extraction candidates
4. Execute decomposition strategy
5. Update dependency graph
6. Synchronize test structure
7. Execute test suite
8. Generate operation report

## DECOMPOSITION STRATEGIES

### Class Extraction
Split multi-class files into single-responsibility units maintaining cohesion boundaries

### Helper Extraction
Extract utility functions into dedicated modules preserving functional purity

### Feature Segregation
Separate distinct business capabilities into isolated modules

### Test Mirroring
Maintain structural isomorphism between source and test directories

## VALIDATION CHECKPOINTS

### File Creation
- Architectural conformance verification
- Line count validation
- Test file generation
- Export barrel update

### File Modification
- Post-edit line count check
- Automatic decomposition trigger
- Import dependency resolution
- Regression test execution

### File Decomposition
- Single Responsibility Principle adherence
- Logical unit extraction
- Cross-reference update
- Test suite synchronization

## AUTOMATED RESPONSES

### Service Generation Trigger
1. Generate service module at designated path
2. Generate repository module at designated path
3. Generate validator module at designated path
4. Create unit test suite
5. Create integration test suite
6. Update barrel exports
7. Validate constraint compliance

### Threshold Violation Trigger
1. Structural analysis execution
2. Boundary identification
3. Decomposition strategy proposal
4. Extraction execution
5. Dependency resolution
6. Test synchronization
7. Verification suite execution

## CONTINUOUS INTEGRATION GATES

### Pre-Commit Validation
- File size constraint verification
- Structural isomorphism validation
- Code coverage threshold enforcement

### Build Pipeline Validation
- Maximum line count compliance
- Test-to-source ratio verification
- Architectural pattern adherence

## ENFORCEMENT COMMANDS

```
npm run lint:file-size
npm run test:validate-structure
npm run test:fix-structure
npm run test:generate-missing
```

## CORE PRINCIPLES

- Line limit enforcement without exception
- Automatic decomposition mandate
- Test parity requirement
- Structural mirroring obligation
- Single Responsibility Principle adherence
- Continuous integration enforcement
- Proactive validation execution

## SUCCESS CRITERIA

- Zero files exceeding constraint
- Complete test-source mirroring
- Zero architectural violations
- Full CI/CD compliance
- Automated enforcement active
- No manual intervention required