import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/core/utils/app_extensions.dart';
import 'package:user_app/app/router/app_navigator.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:user_app/features/settings/presentation/controllers/app_settings_controller.dart';
import 'package:user_app/core/constants/app_constants.dart';
import 'package:user_app/core/constants/app_spacing.dart';

/// Compact theme selection bottom sheet
class ThemeBottomSheet extends ConsumerWidget {
  const ThemeBottomSheet({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final selectedMode = ref.watch(
      appSettingsControllerProvider.select((settings) => settings.themeMode),
    );
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
                AppTranslationKey.chooseTheme.tr,
                style: context.textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
              ),
            ),
            AppSpacing.gapMd,
            Column(
              children: [
                _buildOption(
                  context,
                  ref,
                  selectedMode,
                  ThemeModes.light,
                  AppTranslationKey.lightMode.tr,
                  AppTranslationKey.lightModeDesc.tr,
                  LucideIcons.sun,
                ),
                _buildOption(
                  context,
                  ref,
                  selectedMode,
                  ThemeModes.dark,
                  AppTranslationKey.darkMode.tr,
                  AppTranslationKey.darkModeDesc.tr,
                  LucideIcons.moon,
                ),
                _buildOption(
                  context,
                  ref,
                  selectedMode,
                  ThemeModes.system,
                  AppTranslationKey.systemDefault.tr,
                  AppTranslationKey.systemDefaultDesc.tr,
                  LucideIcons.monitor,
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildOption(
    BuildContext context,
    WidgetRef ref,
    String selectedMode,
    String mode,
    String title,
    String subtitle,
    IconData icon,
  ) {
    final isSelected = selectedMode == mode;
    return ListTile(
      leading: Icon(
        icon,
        color: isSelected ? context.theme.colorScheme.primary : null,
      ),
      title: Text(
        title,
        style: TextStyle(
          fontWeight: isSelected ? FontWeight.w600 : FontWeight.w500,
          color: isSelected ? context.theme.colorScheme.primary : null,
        ),
      ),
      subtitle: Text(subtitle),
      trailing: isSelected
          ? Icon(
              LucideIcons.check,
              color: context.theme.colorScheme.primary,
              size: 20,
            )
          : null,
      onTap: () async {
        await ref
            .read(appSettingsControllerProvider.notifier)
            .setThemeMode(mode);
        AppNavigator.pop();
      },
      contentPadding: AppSpacing.listItemPadding,
    );
  }

  static void show() {
    AppNavigator.bottomSheet(
      child: const ThemeBottomSheet(),
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
