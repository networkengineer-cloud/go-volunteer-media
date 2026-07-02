import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { Mock } from 'vitest';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import AnimalForm from './AnimalForm';
import { animalsApi, animalTagsApi, commentTagsApi, animalCommentsApi } from '../api/client';
import type { AxiosResponse } from 'axios';
import { ToastProvider } from '../contexts/ToastContext';

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
});
