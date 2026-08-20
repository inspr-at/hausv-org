# Dialog Focus Wait Test Fixture

This fixture proves that the wait logic in `energy-redesign-contract.mjs` correctly fails when focus return is missing.

## Running the Fixture

```bash
cd scripts/snapshot
python3 -m http.server 8000
```

Then open: http://localhost:8000/test-dialog-focus-wait.html

## Expected Behavior

### Test 1: Dialog WITH focus return
- Click "Run Test 1"
- **Expected**: ✓ Test passes - Dialog closed AND focus returned
- **Proof**: The wait correctly succeeds when focus is properly restored

### Test 2: Dialog WITHOUT focus return
- Click "Run Test 2"  
- **Expected**: ✗ Test fails within 2 seconds with "Focus not returned (activeElement: body)"
- **Proof**: The wait correctly catches when focus is NOT restored

## Implementation Details

The fixture uses the **exact same wait logic** as the production test:

```javascript
// Wait for BOTH conditions simultaneously
await page.waitForFunction((opener) => {
  const dialog = document.querySelector('#energy-consumer-dialog');
  return !dialog?.open && document.activeElement === opener;
}, openerHandle, { timeout: 5000 });
```

This demonstrates that:
1. The wait doesn't weaken the original assertion
2. The wait properly fails when focus return is omitted
3. The wait succeeds when both conditions are met
4. The timeout provides clear failure evidence on slow runners

## Why This Fix Works

**Before**: Sequential checks created a race
```javascript
await dialog.waitFor({ state: 'hidden' });        // Wait for dialog
await page.waitForFunction(... focus check ...);  // THEN wait for focus
```

**After**: Simultaneous check eliminates the race
```javascript
await page.waitForFunction(() => {
  return !dialog?.open && document.activeElement === opener;  // BOTH together
});
```

The dialog close and focus return happen asynchronously in the native `<dialog>` close handler. By checking both conditions together, we wait until BOTH operations complete, regardless of which finishes first or how slow the runner is.
