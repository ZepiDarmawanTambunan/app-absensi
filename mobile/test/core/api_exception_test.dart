import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:app_absensi/core/network/api_exception.dart';

DioException _responseError(int status, Map<String, dynamic> data) =>
    DioException(
      requestOptions: RequestOptions(path: '/x'),
      response: Response(
        requestOptions: RequestOptions(path: '/x'),
        statusCode: status,
        data: data,
      ),
      type: DioExceptionType.badResponse,
    );

void main() {
  test('unwraps the backend error envelope', () {
    final e = ApiException.fromDio(_responseError(401, {
      'error': {'code': 'unauthorized', 'message': 'Token kedaluwarsa.'}
    }));
    expect(e.code, 'unauthorized');
    expect(e.message, 'Token kedaluwarsa.');
    expect(e.statusCode, 401);
    expect(e.isUnauthorized, isTrue);
  });

  test('maps connection errors to network', () {
    final e = ApiException.fromDio(DioException(
      requestOptions: RequestOptions(path: '/x'),
      type: DioExceptionType.connectionError,
    ));
    expect(e.code, 'network');
    expect(e.isNetworkError, isTrue);
  });

  test('maps timeouts to timeout', () {
    final e = ApiException.fromDio(DioException(
      requestOptions: RequestOptions(path: '/x'),
      type: DioExceptionType.connectionTimeout,
    ));
    expect(e.code, 'timeout');
    expect(e.isNetworkError, isTrue);
  });

  test('maps 423 to locked', () {
    final e = ApiException.fromDio(_responseError(423, {
      'error': {'code': 'locked', 'message': 'Terkunci.'}
    }));
    expect(e.isLocked, isTrue);
  });

  test('falls back gracefully on unknown shapes', () {
    final e = ApiException.fromDio(_responseError(500, {'weird': true}));
    expect(e.code, 'server_error');
    expect(e.statusCode, 500);
  });
}
