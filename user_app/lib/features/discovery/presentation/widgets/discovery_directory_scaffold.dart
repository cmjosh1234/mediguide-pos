import 'package:flutter/material.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:user_app/core/constants/app_spacing.dart';

/// Shared directory layout matching the app's reference and browse pages.
class DiscoveryDirectoryScaffold extends StatelessWidget {
  const DiscoveryDirectoryScaffold({
    super.key,
    required this.title,
    required this.subtitle,
    required this.searchHint,
    required this.search,
    required this.onChanged,
    required this.onSubmitted,
    required this.onClear,
    required this.onRefresh,
    required this.child,
  });

  final String title, subtitle, searchHint;
  final TextEditingController search;
  final ValueChanged<String> onChanged;
  final VoidCallback onSubmitted, onClear;
  final Future<void> Function() onRefresh;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    return Scaffold(
      backgroundColor: colors.surface,
      appBar: AppBar(
        toolbarHeight:
            (MediaQuery.textScalerOf(context).scale(24) +
                    MediaQuery.textScalerOf(context).scale(14) +
                    12)
                .clamp(kToolbarHeight, double.infinity),
        titleSpacing: AppSpacing.md,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: text.titleLarge?.copyWith(fontWeight: FontWeight.w800),
            ),
            Text(
              subtitle,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: text.bodySmall?.copyWith(color: colors.onSurfaceVariant),
            ),
          ],
        ),
      ),
      body: RefreshIndicator(
        onRefresh: onRefresh,
        child: CustomScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
          slivers: [
            SliverPadding(
              padding: const EdgeInsets.all(AppSpacing.md),
              sliver: SliverToBoxAdapter(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    TextField(
                      controller: search,
                      textInputAction: TextInputAction.search,
                      onChanged: onChanged,
                      onSubmitted: (_) => onSubmitted(),
                      decoration: InputDecoration(
                        hintText: searchHint,
                        prefixIcon: const Icon(LucideIcons.search, size: 20),
                        suffixIcon: search.text.isEmpty
                            ? null
                            : IconButton(
                                tooltip: 'Clear search',
                                onPressed: onClear,
                                icon: const Icon(LucideIcons.x),
                              ),
                        filled: true,
                        fillColor: colors.surfaceContainerLow,
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(14),
                          borderSide: BorderSide(color: colors.outlineVariant),
                        ),
                        enabledBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(14),
                          borderSide: BorderSide(color: colors.outlineVariant),
                        ),
                      ),
                    ),
                    AppSpacing.gapLg,
                    child,
                    AppSpacing.gapXxl,
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class DiscoveryDirectoryResults extends StatelessWidget {
  const DiscoveryDirectoryResults({
    super.key,
    required this.label,
    required this.count,
    required this.offline,
    required this.child,
  });
  final String label;
  final int count;
  final bool offline;
  final Widget child;

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      if (offline) ...[
        const Card(
          child: ListTile(
            leading: Icon(LucideIcons.cloudOff),
            title: Text('Showing saved content'),
            subtitle: Text('Pull down to try connecting again.'),
          ),
        ),
        AppSpacing.gapMd,
      ],
      Row(
        children: [
          Expanded(
            child: Text(
              label,
              style: Theme.of(
                context,
              ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800),
            ),
          ),
          const SizedBox(width: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.surfaceContainerLow,
              borderRadius: BorderRadius.circular(20),
            ),
            child: Text(
              '$count',
              style: Theme.of(context).textTheme.labelLarge,
            ),
          ),
        ],
      ),
      AppSpacing.gapSm,
      child,
    ],
  );
}
