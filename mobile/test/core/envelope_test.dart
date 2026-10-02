import 'package:flutter_test/flutter_test.dart';

import 'package:app_absensi/core/network/api_exception.dart';
import 'package:app_absensi/core/network/envelope.dart';

void main() {
  test('unwraps data payloads', () {
    final value = ApiEnvelope.data(
        {'data': {'a': 1}}, (raw) => (raw as Map<String, dynamic>)['a']);
    expect(value, 1);
  });

  test('throws ApiException on error envelopes', () {
    expect(
      () => ApiEnvelope.data(
          {'error': {'code': 'nope', 'message': 'Tidak.'}}, (raw) => raw),
      throwsA(isA<ApiException>()
          .having((e) => e.code, 'code', 'nope')
          .having((e) => e.message, 'message', 'Tidak.')),
    );
  });

  test('throws on malformed payloads', () {
    expect(
      () => ApiEnvelope.data({'surprise': true}, (raw) => raw),
      throwsA(isA<ApiException>().having((e) => e.code, 'code', 'malformed')),
    );
  });

  test('dataList maps arrays', () {
    final list = ApiEnvelope.dataList(
        {'data': [{'n': 1}, {'n': 2}]}, (m) => m['n'] as int);
    expect(list, [1, 2]);
  });
}
