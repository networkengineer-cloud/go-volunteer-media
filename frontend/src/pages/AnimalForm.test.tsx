import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { Mock } from 'vitest';
import { render, screen, waitFor, fireEvent, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import AnimalForm from './AnimalForm';
import { animalsApi, animalTagsApi, commentTagsApi, animalCommentsApi } from '../api/client';
import type { AxiosResponse } from 'axios';
import { ToastProvider } from '../contexts/ToastContext';
import { calculateQuarantineEndDateISO } from '../utils/dateUtils';

// Mock the API client
vi.mock('../api/client', () => ({
  animalsApi: {
    getById: vi.fn(),
    checkDuplicates: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    getImages: vi.fn(),
  },
  animalTagsApi: {
    getAll: vi.fn(),
    assignToAnimal: vi.fn(),
  },
  commentTagsApi: {
    getAll: vi.fn(),
  },
  animalCommentsApi: {
    create: vi.fn(),
  },
}));

// Mock useParams/useNavigate for edit mode
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useParams: () => ({ groupId: '1', id: '1' }),
    useNavigate: () => mockNavigate,
  };
});

const existingAnimal = {
  id: 1,
  group_id: 1,
  name: 'Rex',
  species: 'Dog',
  breed: 'Mixed',
  age: 3,
  description: '',
  image_url: '',
  status: 'available',
  is_returned: false,
};

const existingQuarantinedAnimal = {
  id: 1,
  group_id: 1,
  name: 'Rex',
  species: 'Dog',
  breed: 'Mixed',
  age: 3,
  description: '',
  image_url: '',
  status: 'bite_quarantine',
  quarantine_start_date: '2026-06-01T00:00:00Z',
  quarantine_approval_status: 'requested',
  quarantine_incident_details: 'Bit a volunteer on the hand.',
  is_returned: false,
};

describe('AnimalForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: existingAnimal,
    } as AxiosResponse);

    vi.mocked(animalsApi.checkDuplicates).mockResolvedValue({
      data: { name: 'Rex', count: 0, animals: [], has_duplicates: false },
    } as AxiosResponse);

    vi.mocked(animalsApi.getImages).mockResolvedValue({
      data: [],
    } as AxiosResponse);

    vi.mocked(animalTagsApi.getAll).mockResolvedValue({
      data: [],
    } as AxiosResponse);

    vi.mocked(commentTagsApi.getAll).mockResolvedValue({
      data: [],
    } as AxiosResponse);

    vi.mocked(animalsApi.update).mockResolvedValue({
      data: { ...existingAnimal, id: 1 },
    } as AxiosResponse);

    vi.mocked(animalCommentsApi.create).mockResolvedValue({} as AxiosResponse);
  });

  const renderAnimalForm = () => {
    return render(
      <BrowserRouter>
        <ToastProvider>
          <AnimalForm />
        </ToastProvider>
      </BrowserRouter>
    );
  };

  it('on quarantine submit: sends incident details, keeps the comment, drops the announcement', async () => {
    const user = userEvent.setup();
    renderAnimalForm();

    // Wait for the form to populate from loadAnimal
    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    // Change status to bite_quarantine
    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'bite_quarantine' } });

    // Submit the main form -> opens the quarantine info modal
    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    // Fill in the incident details textarea in the modal
    const incidentTextarea = await screen.findByLabelText(/incident details/i);
    await user.type(incidentTextarea, 'Bit a volunteer.');

    // Submit the modal
    const modalSubmitButton = screen.getByRole('button', { name: /save & notify/i });
    await user.click(modalSubmitButton);

    await waitFor(() => expect(animalsApi.update).toHaveBeenCalled());

    const payload = (animalsApi.update as Mock).mock.calls[0][2];
    expect(payload.quarantine_incident_details).toBe('Bit a volunteer.');
    expect(animalCommentsApi.create).toHaveBeenCalled();
  });

  it('loads existing incident details and start date for an animal already in quarantine', async () => {
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: existingQuarantinedAnimal,
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const incidentTextarea = document.getElementById('quarantine_incident_details') as HTMLTextAreaElement;
    const dateInput = document.getElementById('quarantine_start_date') as HTMLInputElement;

    expect(incidentTextarea.value).toBe('Bit a volunteer on the hand.');
    expect(dateInput.value).toBe('2026-06-01');
  });

  it('saves edited incident details/date for an already-quarantined animal via the normal save flow', async () => {
    const user = userEvent.setup();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: existingQuarantinedAnimal,
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const incidentTextarea = document.getElementById('quarantine_incident_details') as HTMLTextAreaElement;
    await user.clear(incidentTextarea);
    await user.type(incidentTextarea, 'Corrected: bit a staff member, not a volunteer.');

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    await waitFor(() => expect(animalsApi.update).toHaveBeenCalled());

    const payload = (animalsApi.update as Mock).mock.calls[0][2];
    expect(payload.quarantine_incident_details).toBe('Corrected: bit a staff member, not a volunteer.');
    expect(animalCommentsApi.create).not.toHaveBeenCalled();
  });

  it('loads the stored quarantine end date for an animal already in quarantine', async () => {
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_end_date: '2026-06-15T00:00:00Z' },
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const endDateInput = document.getElementById('quarantine_end_date') as HTMLInputElement;
    expect(endDateInput.value).toBe('2026-06-15');
  });

  it('recomputes the end date field when the start date changes', async () => {
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_end_date: '2026-06-15T00:00:00Z' },
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const startDateInput = document.getElementById('quarantine_start_date') as HTMLInputElement;
    fireEvent.change(startDateInput, { target: { value: '2024-06-03' } }); // Monday

    const endDateInput = document.getElementById('quarantine_end_date') as HTMLInputElement;
    expect(endDateInput.value).toBe('2024-06-13'); // 10 days later (Thursday)
  });

  it('keeps a manually edited end date on submit instead of recomputing it', async () => {
    const user = userEvent.setup();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_end_date: '2026-06-11T00:00:00Z' },
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const endDateInput = document.getElementById('quarantine_end_date') as HTMLInputElement;
    fireEvent.change(endDateInput, { target: { value: '2026-06-20' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    await waitFor(() => expect(animalsApi.update).toHaveBeenCalled());

    const payload = (animalsApi.update as Mock).mock.calls[0][2];
    expect(payload.quarantine_end_date).toBe('2026-06-20');
  });

  it('blocks submit with a validation error when the end date is before the start date', async () => {
    const user = userEvent.setup();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_end_date: '2026-06-11T00:00:00Z' },
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const endDateInput = document.getElementById('quarantine_end_date') as HTMLInputElement;
    fireEvent.change(endDateInput, { target: { value: '2026-05-01' } }); // before start date 2026-06-01

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    expect(submitButton).toBeDisabled();

    await user.click(submitButton);

    expect(animalsApi.update).not.toHaveBeenCalled();
  });

  it('defaults the quarantine end date in the new-incident modal and submits it', async () => {
    const user = userEvent.setup();
    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'bite_quarantine' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    const biteDateInput = await screen.findByLabelText(/bite date/i) as HTMLInputElement;
    fireEvent.change(biteDateInput, { target: { value: '2024-06-03' } }); // Monday

    const modalEndDateInput = screen.getByLabelText(/quarantine end date/i) as HTMLInputElement;
    expect(modalEndDateInput.value).toBe('2024-06-13'); // 10 days later (Thursday)

    const incidentTextarea = screen.getByLabelText(/incident details/i);
    await user.type(incidentTextarea, 'Bit a volunteer.');

    const modalSubmitButton = screen.getByRole('button', { name: /save & notify/i });
    await user.click(modalSubmitButton);

    await waitFor(() => expect(animalsApi.update).toHaveBeenCalled());

    const payload = (animalsApi.update as Mock).mock.calls[0][2];
    expect(payload.quarantine_start_date).toBe('2024-06-03');
    expect(payload.quarantine_end_date).toBe('2024-06-13');
  });

  it('submits a vet-overridden end date entered directly in the new-incident modal', async () => {
    const user = userEvent.setup();
    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'bite_quarantine' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    const modalEndDateInput = await screen.findByLabelText(/quarantine end date/i) as HTMLInputElement;
    fireEvent.change(modalEndDateInput, { target: { value: '2026-08-01' } }); // vet-extended

    const incidentTextarea = screen.getByLabelText(/incident details/i);
    await user.type(incidentTextarea, 'Bit a volunteer.');

    const modalSubmitButton = screen.getByRole('button', { name: /save & notify/i });
    await user.click(modalSubmitButton);

    await waitFor(() => expect(animalsApi.update).toHaveBeenCalled());

    const payload = (animalsApi.update as Mock).mock.calls[0][2];
    expect(payload.quarantine_end_date).toBe('2026-08-01');
  });

  it('disables Save & Notify in the new-incident modal when the end date precedes the bite date', async () => {
    const user = userEvent.setup();
    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'bite_quarantine' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    const biteDateInput = await screen.findByLabelText(/bite date/i) as HTMLInputElement;
    fireEvent.change(biteDateInput, { target: { value: '2026-06-10' } });

    const modalEndDateInput = screen.getByLabelText(/quarantine end date/i) as HTMLInputElement;
    fireEvent.change(modalEndDateInput, { target: { value: '2026-06-01' } }); // before bite date

    const incidentTextarea = screen.getByLabelText(/incident details/i);
    await user.type(incidentTextarea, 'Bit a volunteer.');

    const modalSubmitButton = screen.getByRole('button', { name: /save & notify/i });
    expect(modalSubmitButton).toBeDisabled();

    await user.click(modalSubmitButton);
    expect(animalsApi.update).not.toHaveBeenCalled();
  });

  it('disables Save & Notify in the new-incident modal when the bite date is cleared', async () => {
    const user = userEvent.setup();
    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'bite_quarantine' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    const biteDateInput = await screen.findByLabelText(/bite date/i) as HTMLInputElement;
    fireEvent.change(biteDateInput, { target: { value: '' } });

    const incidentTextarea = screen.getByLabelText(/incident details/i);
    await user.type(incidentTextarea, 'Bit a volunteer.');

    const modalSubmitButton = screen.getByRole('button', { name: /save & notify/i });
    expect(modalSubmitButton).toBeDisabled();

    await user.click(modalSubmitButton);
    expect(animalsApi.update).not.toHaveBeenCalled();
  });

  it('re-populates fresh quarantine dates and clears incident details when toggling away from and back to bite_quarantine', async () => {
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_end_date: '2026-06-11T00:00:00Z' },
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    // Toggle away from bite_quarantine, then back to it within the same session
    // (without saving in between) — re-entering should populate fresh, today-based
    // defaults rather than either silently retaining the previous incident's stale
    // dates or leaving the fields blank for saveAnimal to guess at.
    fireEvent.change(statusSelect, { target: { value: 'available' } });
    fireEvent.change(statusSelect, { target: { value: 'bite_quarantine' } });

    const startDateInput = document.getElementById('quarantine_start_date') as HTMLInputElement;
    const endDateInput = document.getElementById('quarantine_end_date') as HTMLInputElement;
    const incidentTextarea = document.getElementById('quarantine_incident_details') as HTMLTextAreaElement;

    const today = new Date().toISOString().split('T')[0];
    expect(startDateInput.value).toBe(today);
    expect(endDateInput.value).toBe(calculateQuarantineEndDateISO(today));
    expect(incidentTextarea.value).toBe('');
  });

  it('saving right after toggling away and back from bite_quarantine sends fresh dates instead of silently keeping stale ones', async () => {
    const user = userEvent.setup();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: {
        ...existingQuarantinedAnimal,
        quarantine_end_date: '2026-06-11T00:00:00Z',
        quarantine_approval_status: 'granted',
      },
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    // Toggle away and back without saving in between, then submit immediately
    // without refilling anything — the save should not silently keep the old
    // (2026-06-01/2026-06-11) dates or the stale 'granted' approval status
    // while wiping incident details out from under them.
    fireEvent.change(statusSelect, { target: { value: 'available' } });
    fireEvent.change(statusSelect, { target: { value: 'bite_quarantine' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    await waitFor(() => expect(animalsApi.update).toHaveBeenCalled());

    const today = new Date().toISOString().split('T')[0];
    const payload = (animalsApi.update as Mock).mock.calls[0][2];
    expect(payload.quarantine_start_date).toBe(today);
    expect(payload.quarantine_end_date).toBe(calculateQuarantineEndDateISO(today));
    expect(payload.quarantine_incident_details).toBe('');
    expect(payload.quarantine_approval_status).toBe('requested');
  });

  it('re-quarantining a previously-archived animal prefills the incident modal with today, not the stale prior incident date', async () => {
    const user = userEvent.setup();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: {
        ...existingQuarantinedAnimal,
        status: 'archived',
        // Archiving doesn't clear these server-side, so a previously-quarantined,
        // now-archived animal still carries its old quarantine stint's data.
        quarantine_start_date: '2024-01-01T00:00:00Z',
        quarantine_end_date: '2024-01-13T00:00:00Z',
        quarantine_approval_status: 'granted',
      },
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'bite_quarantine' } });

    const confirmButton = await screen.findByRole('button', { name: /confirm change/i });
    await user.click(confirmButton);

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    const today = new Date().toISOString().split('T')[0];
    const biteDateInput = await screen.findByLabelText(/bite date/i) as HTMLInputElement;
    expect(biteDateInput.value).toBe(today);

    const modalEndDateInput = screen.getByLabelText(/quarantine end date/i) as HTMLInputElement;
    expect(modalEndDateInput.value).toBe(calculateQuarantineEndDateISO(today));
  });

  it('directly clearing the inline Start/End Date fields on an already-quarantined animal leaves the stored dates untouched instead of fabricating new ones', async () => {
    const user = userEvent.setup();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_end_date: '2026-06-11T00:00:00Z' },
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    // Directly blank the inline Start Date field (status stays bite_quarantine the
    // whole time — no toggling involved). Its own onChange recomputes End Date to
    // '' too. Saving here must NOT silently substitute today's date for the real,
    // previously-recorded quarantine window.
    const startDateInput = document.getElementById('quarantine_start_date') as HTMLInputElement;
    fireEvent.change(startDateInput, { target: { value: '' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    await waitFor(() => expect(animalsApi.update).toHaveBeenCalled());

    const payload = (animalsApi.update as Mock).mock.calls[0][2];
    expect(payload.quarantine_start_date).toBeUndefined();
    expect(payload.quarantine_end_date).toBeUndefined();
  });

  it('shows the BQ exit confirmation modal instead of saving immediately when leaving bite_quarantine', async () => {
    const user = userEvent.setup();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_end_date: '2026-06-15T00:00:00Z' },
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'available' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    expect(await screen.findByText(/confirm bite quarantine exit/i)).toBeInTheDocument();
    expect(animalsApi.update).not.toHaveBeenCalled();
  });

  it('defaults the BQ exit modal to the stored end date when closing out late', async () => {
    const user = userEvent.setup();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_end_date: '2020-01-13T00:00:00Z' }, // long past
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'available' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    const exitEndDateInput = await screen.findByLabelText(/quarantine end date/i) as HTMLInputElement;
    expect(exitEndDateInput.value).toBe('2020-01-13');
  });

  it('defaults the BQ exit modal to today when leaving quarantine early', async () => {
    const user = userEvent.setup();
    const farFutureEndDate = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_end_date: farFutureEndDate },
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'available' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    // Computed via local Date getters, not toISOString() (UTC) — matching
    // production's bqExitDefaultEndDate. Using toISOString() here would make
    // this test pass even if the local-date fix in production regressed back
    // to UTC, since the two only diverge for non-UTC callers.
    const now = new Date();
    const today = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
    const exitEndDateInput = await screen.findByLabelText(/quarantine end date/i) as HTMLInputElement;
    expect(exitEndDateInput.value).toBe(today);
  });

  it('confirming the BQ exit modal submits the status change with the confirmed end date', async () => {
    const user = userEvent.setup();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_end_date: '2026-06-15T00:00:00Z' },
    } as AxiosResponse);
    vi.mocked(animalsApi.update).mockResolvedValue({ data: { id: 1 } } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'available' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    const exitEndDateInput = await screen.findByLabelText(/quarantine end date/i) as HTMLInputElement;
    fireEvent.change(exitEndDateInput, { target: { value: '2026-06-10' } });

    const confirmButton = screen.getByRole('button', { name: /confirm & save/i });
    await user.click(confirmButton);

    await waitFor(() => expect(animalsApi.update).toHaveBeenCalled());

    const payload = (animalsApi.update as Mock).mock.calls[0][2];
    expect(payload.status).toBe('available');
    expect(payload.quarantine_end_date).toBe('2026-06-10');
  });

  it('confirming the BQ exit modal also syncs tag assignments, matching the normal save path', async () => {
    const user = userEvent.setup();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_end_date: '2026-06-15T00:00:00Z' },
    } as AxiosResponse);
    vi.mocked(animalsApi.update).mockResolvedValue({ data: { id: 1 } } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'available' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    await screen.findByLabelText(/quarantine end date/i);

    const confirmButton = screen.getByRole('button', { name: /confirm & save/i });
    await user.click(confirmButton);

    await waitFor(() => expect(animalsApi.update).toHaveBeenCalled());
    expect(animalTagsApi.assignToAnimal).toHaveBeenCalled();
  });

  it('cancelling the BQ exit modal does not save and leaves the form open', async () => {
    const user = userEvent.setup();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_end_date: '2026-06-15T00:00:00Z' },
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'available' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    await screen.findByText(/confirm bite quarantine exit/i);

    // Scoped to the modal dialog: the form's own "Cancel" (navigate away) button
    // is also present in the DOM while the modal is open and shares the same name.
    const dialog = screen.getByRole('dialog', { name: /confirm bite quarantine exit/i });
    const cancelButton = within(dialog).getByRole('button', { name: /^cancel$/i });
    await user.click(cancelButton);

    expect(screen.queryByText(/confirm bite quarantine exit/i)).not.toBeInTheDocument();
    expect(animalsApi.update).not.toHaveBeenCalled();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('blocks confirming the BQ exit modal when the end date precedes the quarantine start date', async () => {
    const user = userEvent.setup();
    vi.mocked(animalsApi.getById).mockResolvedValue({
      data: { ...existingQuarantinedAnimal, quarantine_start_date: '2026-06-01T00:00:00Z', quarantine_end_date: '2026-06-15T00:00:00Z' },
    } as AxiosResponse);

    renderAnimalForm();

    await waitFor(() => {
      const nameInput = document.getElementById('name') as HTMLInputElement;
      expect(nameInput.value).toBe('Rex');
    });

    const statusSelect = document.getElementById('status') as HTMLSelectElement;
    fireEvent.change(statusSelect, { target: { value: 'available' } });

    const submitButton = screen.getByRole('button', { name: /update animal/i });
    await user.click(submitButton);

    const exitEndDateInput = await screen.findByLabelText(/quarantine end date/i) as HTMLInputElement;
    fireEvent.change(exitEndDateInput, { target: { value: '2026-05-01' } }); // before start date

    const confirmButton = screen.getByRole('button', { name: /confirm & save/i });
    expect(confirmButton).toBeDisabled();

    await user.click(confirmButton);
    expect(animalsApi.update).not.toHaveBeenCalled();
  });
});
