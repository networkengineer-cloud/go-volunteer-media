import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import QuarantineApprovalBadge from './QuarantineApprovalBadge';

describe('QuarantineApprovalBadge', () => {
  it('renders "Not Requested" state for null status', () => {
    const { container } = render(<QuarantineApprovalBadge status={null} />);
    expect(screen.getByText(/Not Requested/)).toBeInTheDocument();
    expect(container.querySelector('[aria-hidden="true"]')).toHaveTextContent('⬜');
  });

  it('renders "Not Requested" state for undefined status', () => {
    render(<QuarantineApprovalBadge />);
    expect(screen.getByText(/Not Requested/)).toBeInTheDocument();
  });

  it('renders "Not Requested" state for empty string status', () => {
    render(<QuarantineApprovalBadge status="" />);
    expect(screen.getByText(/Not Requested/)).toBeInTheDocument();
    expect(screen.getByLabelText('Bite Quarantine Permission: Not Requested')).toBeInTheDocument();
  });

  it('renders "Requested" state with correct aria-label', () => {
    render(<QuarantineApprovalBadge status="requested" />);
    expect(screen.getByText(/Requested/)).toBeInTheDocument();
    expect(screen.getByLabelText('Bite Quarantine Permission: Requested — Awaiting Response')).toBeInTheDocument();
  });

  it('renders "Granted — Cleared to Work" state with correct aria-label', () => {
    render(<QuarantineApprovalBadge status="granted" />);
    expect(screen.getByText(/Granted — Cleared to Work/)).toBeInTheDocument();
    expect(screen.getByLabelText('Bite Quarantine Permission: Granted — Cleared to Work')).toBeInTheDocument();
  });

  it('applies quarantine-approval-none class for null status', () => {
    render(<QuarantineApprovalBadge status={null} />);
    const badge = screen.getByLabelText('Bite Quarantine Permission: Not Requested');
    expect(badge).toHaveClass('quarantine-approval-badge');
    expect(badge).toHaveClass('quarantine-approval-none');
  });

  it('applies quarantine-approval-requested class', () => {
    render(<QuarantineApprovalBadge status="requested" />);
    const badge = screen.getByLabelText('Bite Quarantine Permission: Requested — Awaiting Response');
    expect(badge).toHaveClass('quarantine-approval-requested');
  });

  it('applies quarantine-approval-granted class', () => {
    render(<QuarantineApprovalBadge status="granted" />);
    const badge = screen.getByLabelText('Bite Quarantine Permission: Granted — Cleared to Work');
    expect(badge).toHaveClass('quarantine-approval-granted');
  });
});
