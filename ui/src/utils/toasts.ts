import { Notyf } from 'notyf';

const notyf = new Notyf({
  position: {
    x: 'center',
    y: 'top',
  },
  ripple: false,
  duration: 5000,
});

export function showToast(message: string, type: 'success' | 'error' = 'error', duration: number = 5000) {
  if (type === 'success') {
    notyf.success({ message, duration });
  } else {
    notyf.error({ message, duration });
  }
}
