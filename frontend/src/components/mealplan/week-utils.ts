// Pure date helpers for the meal-plan week view.
//
// The Go domain owns the canonical WeekStartOf (internal/domain/mealplan);
// the frontend keeps a tiny mirror so it can compute weekStart without a
// round trip to Go on mount, and generate the 7-day column headers.

export function formatDate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

export function weekStartOf(d: Date): Date {
  const day = d.getDay(); // Sunday=0, Monday=1, ..., Saturday=6
  const offset = (day + 6) % 7; // distance back to Monday
  const monday = new Date(d.getFullYear(), d.getMonth(), d.getDate() - offset);
  return monday;
}

export function weekDates(weekStart: Date): Date[] {
  return Array.from({ length: 7 }, (_, i) => {
    return new Date(weekStart.getFullYear(), weekStart.getMonth(), weekStart.getDate() + i);
  });
}

export function isSameLocalDate(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}
