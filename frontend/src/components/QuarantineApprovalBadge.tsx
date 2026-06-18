import React from 'react';

interface Props {
  status?: string | null;
}

const QuarantineApprovalBadge: React.FC<Props> = ({ status }) => {
  const normalised = status || 'none';

  const ariaLabel =
    status === 'granted'
      ? 'Bite Quarantine Permission: Granted — Cleared to Work'
      : status === 'requested'
      ? 'Bite Quarantine Permission: Requested — Awaiting Response'
      : 'Bite Quarantine Permission: Not Requested';

  const emoji =
    status === 'granted' ? '✅' : status === 'requested' ? '🕐' : '⬜';

  const label =
    status === 'granted'
      ? 'Granted — Cleared to Work'
      : status === 'requested'
      ? 'Requested'
      : 'Not Requested';

  return (
    <span
      className={`quarantine-approval-badge quarantine-approval-${normalised}`}
      aria-label={ariaLabel}
    >
      <span aria-hidden="true">{emoji}</span>
      {' '}{label}
    </span>
  );
};

export default QuarantineApprovalBadge;
