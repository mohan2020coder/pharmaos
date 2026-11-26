export const colors = {
  primary: '#1D4ED8',
  primaryDark: '#16324F',
  primarySoft: '#EFF6FF',
  surface: '#FFFFFF',
  background: '#F7F9FC',
  text: '#0F172A',
  muted: '#64748B',
  border: '#CBD5E1',
  borderSoft: '#E2E8F0',
  danger: '#B91C1C',
  dangerSoft: '#FEF2F2',
  success: '#047857',
  successSoft: '#ECFDF5',
  inverse: '#FFFFFF',
  overlay: 'rgba(15, 23, 42, 0.55)'
} as const;

export const spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
  xxxl: 48
} as const;

export const radius = {
  sm: 8,
  md: 12,
  lg: 16,
  pill: 999
} as const;

export const typography = {
  display: { fontSize: 34, fontWeight: '700', lineHeight: 40 },
  title: { fontSize: 26, fontWeight: '700', lineHeight: 32 },
  heading: { fontSize: 20, fontWeight: '700', lineHeight: 26 },
  section: { fontSize: 16, fontWeight: '600', lineHeight: 22 },
  body: { fontSize: 15, fontWeight: '400', lineHeight: 21 },
  caption: { fontSize: 12, fontWeight: '500', lineHeight: 16 },
  button: { fontSize: 16, fontWeight: '600', lineHeight: 22 }
} as const;

export const shadows = {
  card: {
    shadowColor: '#0F172A',
    shadowOpacity: 0.08,
    shadowRadius: 12,
    shadowOffset: { width: 0, height: 4 },
    elevation: 3
  }
} as const;
