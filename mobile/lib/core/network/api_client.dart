import 'dart:async';

import 'package:dio/dio.dart';

import '../models/token_pair.dart';
import '../storage/secure_token_storage.dart';
import 'api_exception.dart';

/// Single HTTP gateway for the whole app.
///
/// - Attaches `Authorization: Bearer <access_token>` to every request.
/// - On 401, refreshes the token pair once (serialized across concurrent
///   requests) and retries the original request.
/// - Throws [ApiException] for every failure — never raw [DioException].
///
/// No widget or bloc may create its own Dio / http client; repositories
/// receive this instance via constructor injection.
class ApiClient {
  final Dio _dio;
  final Dio _refreshDio;
  final SecureTokenStorage _storage;

  bool _refreshing = false;
  Completer<void>? _refreshCompleter;

  ApiClient({
    required String baseUrl,
    required SecureTokenStorage storage,
    Dio? dio,
  })  : _storage = storage,
        _dio = dio ??
            Dio(BaseOptions(
              baseUrl: baseUrl,
              connectTimeout: const Duration(seconds: 15),
              receiveTimeout: const Duration(seconds: 30),
              contentType: 'application/json',
              responseType: ResponseType.json,
            )),
        _refreshDio = Dio(BaseOptions(
          baseUrl: baseUrl,
          contentType: 'application/json',
          responseType: ResponseType.json,
        )) {
    _dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) async {
        final token = await _storage.accessToken;
        if (token != null && token.isNotEmpty) {
          options.headers['Authorization'] = 'Bearer $token';
        }
        handler.next(options);
      },
      onError: (error, handler) async {
        final req = error.requestOptions;
        final isRefreshCall = req.path.contains('/auth/refresh');
        final alreadyRetried = req.extra['retried'] == true;
        if (error.response?.statusCode == 401 &&
            !isRefreshCall &&
            !alreadyRetried) {
          try {
            await _refreshTokens();
            final token = await _storage.accessToken;
            req.extra['retried'] = true;
            if (token != null && token.isNotEmpty) {
              req.headers['Authorization'] = 'Bearer $token';
            }
            final response = await _dio.fetch<Map<String, dynamic>>(req);
            handler.resolve(response);
            return;
          } catch (_) {
            // Fall through: report the original 401 below.
          }
        }
        handler.next(error);
      },
    ));
  }

  /// Serializes concurrent refresh attempts into a single token refresh.
  Future<void> _refreshTokens() async {
    if (_refreshing) {
      final completer = _refreshCompleter;
      if (completer != null) await completer.future;
      return;
    }
    _refreshing = true;
    _refreshCompleter = Completer<void>();
    try {
      final refreshToken = await _storage.refreshToken;
      if (refreshToken == null || refreshToken.isEmpty) {
        throw const ApiException(
            code: 'no_refresh_token', message: 'Sesi berakhir. Silakan login kembali.');
      }
      final res = await _refreshDio.post<Map<String, dynamic>>(
        '/auth/refresh',
        data: {'refresh_token': refreshToken},
      );
      final pair = TokenPair.fromEnvelope(res.data ?? {});
      await _storage.saveTokens(
        accessToken: pair.accessToken,
        refreshToken: pair.refreshToken,
      );
      _refreshCompleter!.complete();
    } catch (e) {
      await _storage.clear();
      _refreshCompleter!.completeError(e);
      rethrow;
    } finally {
      _refreshing = false;
      _refreshCompleter = null;
    }
  }

  Future<Map<String, dynamic>> get(
    String path, {
    Map<String, dynamic>? query,
  }) async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>(path, queryParameters: query);
      return res.data ?? {};
    } on DioException catch (e) {
      throw ApiException.fromDio(e);
    }
  }

  Future<Map<String, dynamic>> post(
    String path, {
    Object? data,
    Map<String, dynamic>? query,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(path,
          data: data, queryParameters: query);
      return res.data ?? {};
    } on DioException catch (e) {
      throw ApiException.fromDio(e);
    }
  }
}
