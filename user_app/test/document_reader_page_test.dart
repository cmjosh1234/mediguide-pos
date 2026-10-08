import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:user_app/features/documents/presentation/screens/document_reader_page.dart';

void main() {
  test('only a file with a PDF header counts as a PDF', () async {
    final directory = await Directory.systemTemp.createTemp('reader-test');
    addTearDown(() => directory.delete(recursive: true));
    Future<File> write(String name, String content) =>
        File('${directory.path}/$name').writeAsString(content);

    expect(await isPdfFile(await write('report.pdf', '%PDF-1.7\n...')), isTrue);
    // A short preamble before the header is allowed.
    expect(
      await isPdfFile(await write('preamble.pdf', '\n\n%PDF-1.4\n...')),
      isTrue,
    );
    // WHO's /bitstreams/…/download link returns its web app instead.
    expect(
      await isPdfFile(
        await write('page.pdf', '<!DOCTYPE html><html><head><title>DSpace'),
      ),
      isFalse,
    );
  });

  test('document reader accepts only supported local and web sources', () {
    expect(
      isSupportedDocumentSource('https://example.test/report.pdf'),
      isTrue,
    );
    expect(isSupportedDocumentSource('http://localhost/report.pdf'), isTrue);
    expect(isSupportedDocumentSource('file:///tmp/report.pdf'), isTrue);
    expect(isSupportedDocumentSource('javascript:alert(1)'), isFalse);
    expect(
      isSupportedDocumentSource('data:application/pdf;base64,abc'),
      isFalse,
    );
    expect(isSupportedDocumentSource(''), isFalse);
  });

  testWidgets('document reader presents a safe designed invalid-source state', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: DocumentReaderPage(
          args: DocumentReaderArgs(
            title: 'Clinical guideline',
            source: 'javascript:alert(1)',
          ),
        ),
      ),
    );

    expect(find.text('Clinical guideline'), findsOneWidget);
    expect(find.text('Document unavailable'), findsOneWidget);
    expect(
      find.text('This document does not have a valid source.'),
      findsOneWidget,
    );
  });
}
