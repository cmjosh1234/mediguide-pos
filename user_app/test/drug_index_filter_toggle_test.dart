import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:infinite_scroll_pagination/infinite_scroll_pagination.dart';
import 'package:user_app/features/drugs/presentation/controllers/drug_index_controller.dart';
import 'package:user_app/features/drugs/presentation/controllers/drug_index_state.dart';
import 'package:user_app/shared/models/models.dart';

class _FilterController extends DrugIndexController {
  @override
  DrugIndexState build() {
    pagingController = PagingController<int, Drug>(
      getNextPageKey: (_) => null,
      fetchPage: (_) async => [],
    );
    ref.onDispose(pagingController.dispose);
    return const DrugIndexState();
  }
}

void main() {
  test('drug filter toggles retain and remove the selected string values', () {
    final container = ProviderContainer(
      overrides: [
        drugIndexControllerProvider.overrideWith(_FilterController.new),
      ],
    );
    addTearDown(container.dispose);
    final subscription = container.listen(
      drugIndexControllerProvider,
      (_, _) {},
    );
    addTearDown(subscription.close);
    final controller = container.read(drugIndexControllerProvider.notifier);

    void toggleAll() {
      controller.toggleCategory('Antibiotics');
      controller.toggleTag('Essential');
      controller.toggleRoute('oral');
      controller.togglePregnancyCategory('B');
    }

    toggleAll();
    var query = container.read(drugIndexControllerProvider).query;
    expect(query.selectedCategories, ['Antibiotics']);
    expect(query.selectedTags, ['Essential']);
    expect(query.selectedRoutes, ['oral']);
    expect(query.selectedPregnancyCategories, ['B']);
    toggleAll();
    query = container.read(drugIndexControllerProvider).query;
    expect(query.selectedCategories, isEmpty);
    expect(query.selectedTags, isEmpty);
    expect(query.selectedRoutes, isEmpty);
    expect(query.selectedPregnancyCategories, isEmpty);
  });
}
