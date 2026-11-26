export function formatDate(value?: string): string {
  if (!value) {
    return '—';
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleDateString();
}

export function formatDateTime(value?: string): string {
  if (!value) {
    return '—';
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export function toDateInput(value: Date | string): string {
  const date = typeof value === 'string' ? new Date(value) : value;
  if (Number.isNaN(date.getTime())) {
    return '';
  }
  return date.toISOString().slice(0, 10);
}

export function todayInput(): string {
  return toDateInput(new Date());
}

export function daysAgoInput(days: number): string {
  const date = new Date();
  date.setDate(date.getDate() - days);
  return toDateInput(date);
}

export function toRFC3339(dateInput: string, timeInput = '09:00'): string | undefined {
  if (!dateInput) {
    return undefined;
  }
  return new Date(`${dateInput}T${timeInput}:00`).toISOString();
}

export function formatMoney(value: number | undefined, currency = 'USD'): string {
  if (value === undefined) {
    return '—';
  }
  return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(value);
}
