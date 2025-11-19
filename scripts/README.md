# Test Scripts

This directory contains helper scripts for testing and validating the codebase before commits and pushes.

## check-offline.sh

Run this **before committing** to validate code quality and functionality without requiring the monitor service.

### Checks performed:
- Code formatting (`make fmt`)
- Linter validation (`make lint`)
- Unit tests (`make test`)
- Build all binaries (`make build`)
- Dependency management (`go mod tidy`)

### Usage:
```bash
./scripts/check-offline.sh
```

### Example output:
```
=== Running offline pre-commit checks ===

1. Checking code formatting...
✓ PASSED: Code formatting

2. Running linter...
✓ PASSED: Linter checks

3. Running unit tests...
✓ PASSED: Unit tests

4. Building binaries...
✓ PASSED: Build

5. Checking go.mod...
✓ PASSED: Dependencies

=== All offline checks passed! ===
You're ready to commit.
```

## check-online.sh

Run this **before pushing** to validate the complete system end-to-end with the monitor service running.

### Prerequisites:
- Monitor service must be running: `./monitor --config config/config.yaml`
- Wait at least 30 seconds for collectors to run and gather metrics

### Checks performed:
- Monitor API is running and healthy
- Health endpoint responds correctly
- Metrics API endpoint is working
- Alert rules are loaded from config
- CLI can query account balances
- CLI network status works
- Metrics are being collected

### Usage:
```bash
# In one terminal, start the monitor
./monitor --config config/config.yaml

# In another terminal, run the online checks
./scripts/check-online.sh
```

### Example output:
```
=== Running online pre-push checks ===
Note: This requires the monitor service to be running

1. Checking if monitor service is running...
✓ PASSED: Monitor service is running

2. Checking health endpoint...
✓ PASSED: Health check

3. Checking metrics API endpoint...
✓ PASSED: Metrics endpoint

4. Checking alert rules...
✓ PASSED: Alert rules loaded (4 rules)

5. Testing CLI account balance query...
✓ PASSED: CLI balance query works

6. Testing CLI network status...
✓ PASSED: CLI network status works

7. Checking if metrics are being collected...
✓ PASSED: Metrics are being collected (47 metrics)

=== Online checks complete ===
Ready to push!
```

## Typical workflow

1. **Before committing:**
   ```bash
   ./scripts/check-offline.sh
   git add .
   git commit -m "Your message"
   ```

2. **Before pushing:**
   ```bash
   # Terminal 1: Start the monitor
   ./monitor --config config/config.yaml

   # Terminal 2: Run online checks
   ./scripts/check-online.sh

   # If all checks pass:
   git push
   ```

## Customization

### change API URL for online checks:
```bash
API_URL=http://monitoring-server.example.com:8080 ./scripts/check-online.sh
```

### Test a specific account:
```bash
TEST_ACCOUNT=0.0.6703786 ./scripts/check-online.sh
```

## Troubleshooting

**Offline checks failing:**
- Run `make fmt` and `make lint` to see detailed errors
- Run `go test ./...` to see which tests are failing

**Online checks failing:**
- Ensure monitor is running: `ps aux | grep monitor`
- Restart monitor: `pkill -f ./monitor && ./monitor --config config/config.yaml`
- Check monitor logs for errors
- Wait 30+ seconds for initial metric collection
- Verify config.yaml has valid credentials

**Monitor not collecting metrics:**
- Check that operator credentials are correct in config.yaml
- Check monitor logs for network/SDK errors
- Verify testnet connectivity
