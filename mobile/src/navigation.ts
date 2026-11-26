import type { SessionUser } from './types';

export type RootStackParamList = {
  Login: undefined;
  Home: { user: SessionUser };
  VisitForm: undefined;
  OrderForm: undefined;
};
