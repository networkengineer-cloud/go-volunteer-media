import React from 'react';

interface Props {
  status?: string | null;
}

const STATUS_CONFIG: Record<string, { ariaLabel: string; emoji: string; label: string }> = {
  granted: {
    ariaLabel: 'Bite Quarantine Permission: Granted — Cleared to Work',
    emoji: '✅',
    label: 'Granted — Cleared to Work',
  },
  requested: {
    ariaLabel: 'Bite Quarantine Permission: Requested — Awaiting Response',
    emoji: '🕐',
    label: 'Requested',
  },
  none: {
    ariaLabel: 'Bite Quarantine Permission: Not Requested',
    emoji: '⬜',
    label: 'Not Requested',
  },
};

const QuarantineApprovalBadge: React.FC<Props> = ({ status }) => {
  const key = status && STATUS_CONFIG[status] ? status : 'none';
  const { ariaLabel, emoji, label } = STATUS_CONFIG[key];

  return (
    <span
      className={`quarantine-approval-badge quarantine-approval-${key}`}
      aria-label={ariaLabel}
    >
      <span aria-hidden="true">{emoji}</span>
      {' '}{label}
    </span>
  );
};

export default QuarantineApprovalBadge;
