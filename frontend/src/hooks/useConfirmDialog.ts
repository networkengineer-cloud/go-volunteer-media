import { useState } from 'react';

interface ConfirmDialogState {
  isOpen: boolean;
  title: string;
  message: string;
  onConfirm: () => void | Promise<void>;
}

const CLOSED: ConfirmDialogState = { isOpen: false, title: '', message: '', onConfirm: () => {} };

export function useConfirmDialog() {
  const [confirmDialog, setConfirmDialog] = useState<ConfirmDialogState>(CLOSED);

  const openConfirmDialog = (
    title: string,
    message: string,
    onConfirm: () => void | Promise<void>,
  ) => setConfirmDialog({ isOpen: true, title, message, onConfirm });

  const closeConfirmDialog = () => setConfirmDialog(d => ({ ...d, isOpen: false }));

  return { confirmDialog, openConfirmDialog, closeConfirmDialog };
}
