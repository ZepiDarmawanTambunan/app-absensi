import 'package:flutter_test/flutter_test.dart';

import 'package:app_absensi/core/storage/key_value_store.dart';
import 'package:app_absensi/core/sync/pending_queue.dart';

void main() {
  test('enqueue / all / remove roundtrip', () async {
    final queue = PendingQueue(InMemoryStore());

    expect(await queue.all(), isEmpty);

    final action = PendingAction(
      id: '1',
      kind: 'check_in',
      employeeNo: 'EMP001',
      recordedAt: DateTime.utc(2026, 10, 2, 8),
    );
    await queue.enqueue(action);

    final items = await queue.all();
    expect(items, hasLength(1));
    expect(items.first, action);

    await queue.remove('1');
    expect(await queue.all(), isEmpty);
  });

  test('actions survive a JSON round-trip', () async {
    final store = InMemoryStore();
    final queue = PendingQueue(store);
    await queue.enqueue(PendingAction(
      id: 'a',
      kind: 'check_out',
      employeeNo: 'EMP002',
      recordedAt: DateTime.utc(2026, 10, 2, 9, 30),
      attempts: 2,
    ));
    // A fresh queue over the same store sees the same actions, proving the
    // JSON serialization (used by the SharedPreferences-backed store) is
    // lossless.
    final reread = PendingQueue(store);
    final items = await reread.all();
    expect(items.single.attempts, 2);
    expect(items.single.employeeNo, 'EMP002');
    expect(items.single.recordedAt, DateTime.utc(2026, 10, 2, 9, 30));
  });
}
