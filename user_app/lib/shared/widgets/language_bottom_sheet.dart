import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/core/utils/app_extensions.dart';
import 'package:user_app/core/widgets/app_skeleton.dart';
import 'package:user_app/app/router/app_navigator.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import 'package:user_app/features/settings/presentation/controllers/language_controller.dart';
import 'package:user_app/core/constants/app_spacing.dart';

class LanguageBottomSheet extends ConsumerWidget {
  const LanguageBottomSheet({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final languages = ref.watch(languageControllerProvider);
    // The theme's bottom sheet supplies the drag handle. Scrolling keeps the
    // options reachable when large text makes them taller than the sheet.
    return SafeArea(
      child: SingleChildScrollView(
        padding: AppSpacing.paddingLg.copyWith(top: 0),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Align(
              alignment: Alignment.centerLeft,
              child: Text(
                AppTranslationKey.chooseLanguage.tr,
                style: context.textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
              ),
            ),
            AppSpacing.gapMd,
            languages.when(
              loading: () => const Padding(
                padding: EdgeInsets.symmetric(vertical: AppSpacing.md),
                child: AppShimmer(
                  child: Column(
                    children: [
                      _LanguageSkeletonRow(),
                      AppSpacing.gapSm,
                      _LanguageSkeletonRow(),
                      AppSpacing.gapSm,
                      _LanguageSkeletonRow(),
                    ],
                  ),
                ),
              ),
              error: (error, stackTrace) => Center(
                child: Padding(
                  padding: EdgeInsets.symmetric(vertical: AppSpacing.lg),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Text('Unable to load languages'),
                      TextButton(
                        onPressed: () => ref
                            .read(languageControllerProvider.notifier)
                            .refresh(),
                        child: const Text('Retry'),
                      ),
                    ],
                  ),
                ),
              ),
              data: (state) => state.languages.isEmpty
                  ? Center(
                      child: Padding(
                        padding: EdgeInsets.symmetric(vertical: AppSpacing.lg),
                        child: const Text('No languages available'),
                      ),
                    )
                  : Column(
                      children: state.languages
                          .map(
                            (language) => _LanguageOption(
                              languageCode: language.code,
                              title: language.name,
                              subtitle: language.nativeName.isNotEmpty
                                  ? language.nativeName
                                  : language.name,
                              selected: state.currentCode == language.code,
                            ),
                          )
                          .toList(),
                    ),
            ),
          ],
        ),
      ),
    );
  }

  static void show() {
    AppNavigator.bottomSheet(
      child: const LanguageBottomSheet(),
      backgroundColor: AppNavigator.theme.colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      clipBehavior: Clip.antiAliasWithSaveLayer,
      isScrollControlled: true,
      enableDrag: true,
      isDismissible: true,
    );
  }
}

class _LanguageSkeletonRow extends StatelessWidget {
  const _LanguageSkeletonRow();

  @override
  Widget build(BuildContext context) => const SizedBox(
    height: 48,
    child: Row(
      children: [
        ClipOval(child: AppSkeleton(width: 36, height: 36)),
        AppSpacing.hGapMd,
        Expanded(child: AppSkeleton(height: 16)),
      ],
    ),
  );
}

class _LanguageOption extends ConsumerWidget {
  const _LanguageOption({
    required this.languageCode,
    required this.title,
    required this.subtitle,
    required this.selected,
  });

  final String languageCode;
  final String title;
  final String subtitle;
  final bool selected;

  @override
  Widget build(BuildContext context, WidgetRef ref) => ListTile(
    leading: Icon(
      LucideIcons.languages,
      color: selected ? context.theme.colorScheme.primary : null,
    ),
    title: Text(
      title,
      style: TextStyle(
        fontWeight: selected ? FontWeight.w600 : FontWeight.w500,
        color: selected ? context.theme.colorScheme.primary : null,
      ),
    ),
    subtitle: Text(subtitle),
    trailing: selected
        ? Icon(
            LucideIcons.check,
            color: context.theme.colorScheme.primary,
            size: 20,
          )
        : null,
    onTap: () async {
      await ref
          .read(languageControllerProvider.notifier)
          .setLanguage(languageCode);
      AppNavigator.pop();
    },
    contentPadding: AppSpacing.listItemPadding,
  );
}
