part of '../screens/use_calculator_page.dart';

class _CalculatorContextBar extends StatelessWidget {
  const _CalculatorContextBar({required this.calculator});
  final Calculator calculator;

  static TextStyle? _titleStyle(BuildContext context) => Theme.of(
    context,
  ).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700, height: 1.3);

  static String _metadata(Calculator calculator) => [
    calculator.type.label,
    if (calculator.version.isNotEmpty) 'v${calculator.version}',
  ].join(' · ');

  // Match the title's wrapping so large accessibility text keeps both lines
  // readable without a fixed-height toolbar clipping the header.
  static double height(BuildContext context, Calculator calculator) {
    final width = (MediaQuery.sizeOf(context).width - 120).clamp(
      1.0,
      double.infinity,
    );
    double measure(String value, TextStyle? style) {
      final painter = TextPainter(
        text: TextSpan(text: value, style: style),
        textDirection: Directionality.of(context),
        textScaler: MediaQuery.textScalerOf(context),
      )..layout(maxWidth: width);
      final height = painter.height;
      painter.dispose();
      return height;
    }

    return (measure(calculator.name, _titleStyle(context)) +
            measure(
              _metadata(calculator),
              Theme.of(context).textTheme.bodySmall,
            ) +
            20)
        .clamp(64.0, double.infinity);
  }

  @override
  Widget build(BuildContext context) => Column(
    mainAxisSize: MainAxisSize.min,
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Text(calculator.name, style: _titleStyle(context)),
      const SizedBox(height: 4),
      Text(
        _metadata(calculator),
        style: Theme.of(context).textTheme.bodySmall?.copyWith(
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      ),
    ],
  );
}
