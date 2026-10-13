/** A message for a failed hub request that says what to do next. */
export function actionable(value: unknown) {
  const message = value instanceof Error ? value.message : "Try again.";
  return /modified|conflict|lock/i.test(message)
    ? "Another administrator changed this hub. Reload it before retrying."
    : /not empty/i.test(message)
      ? "Templates can only be applied to an empty draft hub. Remove existing pillars or create a new hub."
      : /not allowed|unsafe/i.test(message)
        ? "Use an approved HTTPS domain or a supported internal application route."
        : message;
}
