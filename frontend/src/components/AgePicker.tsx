import { useState, useCallback, useEffect } from 'react';
import { calculateAge, computeEstimatedBirthDate } from '../utils/dateUtils';
import './AgePicker.css';

interface AgePickerProps {
  years: number;
  months: number;
  exactDate: string; // YYYY-MM-DD or ''
  useExactDate: boolean;
  onChange: (years: number, months: number, exactDate: string, useExactDate: boolean) => void;
  error?: string;
}

export default function AgePicker({ years, months, exactDate, useExactDate, onChange, error }: AgePickerProps) {
  const [showExactDate, setShowExactDate] = useState(useExactDate);

  // Sync external useExactDate prop
  useEffect(() => {
    setShowExactDate(useExactDate);
  }, [useExactDate]);

  const handleYearsChange = useCallback((val: string) => {
    const y = Math.max(0, Math.min(30, parseInt(val) || 0));
    const computedDate = computeEstimatedBirthDate(y, months);
    onChange(y, months, computedDate, false);
  }, [months, onChange]);

  const handleMonthsChange = useCallback((val: string) => {
    const m = Math.max(0, Math.min(11, parseInt(val) || 0));
    const computedDate = computeEstimatedBirthDate(years, m);
    onChange(years, m, computedDate, false);
  }, [years, onChange]);

  const handleExactDateChange = useCallback((val: string) => {
    if (val) {
      const { years: y, months: m } = calculateAge(val);
      onChange(y, m, val, true);
    } else {
      onChange(years, months, '', true);
    }
  }, [years, months, onChange]);

  const toggleExactDate = useCallback(() => {
    const newShow = !showExactDate;
    setShowExactDate(newShow);
    if (newShow && !exactDate && (years > 0 || months > 0)) {
      // Switching to exact date mode — pre-compute a date from years/months
      const computedDate = computeEstimatedBirthDate(years, months);
      onChange(years, months, computedDate, true);
    } else if (!newShow && exactDate) {
      // Switching back to approximate — recompute years/months from existing date
      const { years: y, months: m } = calculateAge(exactDate);
      onChange(y, m, exactDate, false);
    } else {
      onChange(years, months, exactDate, newShow);
    }
  }, [showExactDate, exactDate, years, months, onChange]);

  const today = new Date().toISOString().split('T')[0];

  return (
    <div className="age-picker">
      <label className="form-field__label">Age</label>

      {!showExactDate ? (
        <>
          <div className="age-picker__inputs">
            <div className="age-picker__field">
              <label htmlFor="birth-years" className="age-picker__sub-label">Years</label>
              <input
                id="birth-years"
                type="number"
                min={0}
                max={30}
                value={years}
                onChange={(e) => handleYearsChange(e.target.value)}
                className={`form-field__input age-picker__number ${error ? 'form-field__input--error' : ''}`}
                aria-label="Age in years"
              />
            </div>
            <div className="age-picker__field">
              <label htmlFor="birth-months" className="age-picker__sub-label">Months</label>
              <input
                id="birth-months"
                type="number"
                min={0}
                max={11}
                value={months}
                onChange={(e) => handleMonthsChange(e.target.value)}
                className={`form-field__input age-picker__number ${error ? 'form-field__input--error' : ''}`}
                aria-label="Age in months"
              />
            </div>
          </div>
          <p className="form-field__helper">
            Approximate age — the birth day will be set to today's date
          </p>
        </>
      ) : (
        <>
          <input
            id="estimated-birth-date"
            type="date"
            value={exactDate}
            onChange={(e) => handleExactDateChange(e.target.value)}
            max={today}
            className={`form-field__input ${error ? 'form-field__input--error' : ''}`}
            aria-label="Estimated birth date"
          />
          <p className="form-field__helper">
            Exact or estimated date of birth
          </p>
        </>
      )}

      {error && <p className="form-field__error">{error}</p>}

      <button
        type="button"
        className="age-picker__toggle"
        onClick={toggleExactDate}
      >
        {showExactDate ? '← Use approximate age instead' : 'Know the exact birth date? →'}
      </button>
    </div>
  );
}
