import { useCallback, useMemo } from 'react';
import { calculateAge, computeEstimatedBirthDate } from '../utils/dateUtils';
import './AgePicker.css';

interface AgePickerProps {
  years: number;
  months: number;
  exactDate: string; // YYYY-MM-DD or ''
  useExactDate: boolean;
  onChange: (years: number, months: number, exactDate: string, useExactDate: boolean) => void;
  error?: string;
  /** Prefix for input element IDs — must be unique per page when the component is used more than once. Defaults to 'age-picker'. */
  idPrefix?: string;
}

export default function AgePicker({ years, months, exactDate, useExactDate, onChange, error, idPrefix = 'age-picker' }: AgePickerProps) {
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
    const newShow = !useExactDate;
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
  }, [useExactDate, exactDate, years, months, onChange]);

  const today = useMemo(() => new Date().toISOString().split('T')[0], []);

  const yearsId = `${idPrefix}-years`;
  const monthsId = `${idPrefix}-months`;
  const dateId = `${idPrefix}-date`;

  return (
    <div className="age-picker">
      <label className="form-field__label" htmlFor={useExactDate ? dateId : yearsId}>Age</label>

      {!useExactDate ? (
        <>
          <div className="age-picker__inputs">
            <div className="age-picker__field">
              <label htmlFor={yearsId} className="age-picker__sub-label">Years</label>
              <input
                id={yearsId}
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
                value={years}
                onChange={(e) => handleYearsChange(e.target.value)}
                className={`form-field__input age-picker__number ${error ? 'form-field__input--error' : ''}`}
                aria-label="Age in years"
              />
            </div>
            <div className="age-picker__field">
              <label htmlFor={monthsId} className="age-picker__sub-label">Months</label>
              <input
                id={monthsId}
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
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
            id={dateId}
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
        {useExactDate ? '← Use approximate age instead' : 'Know the exact birth date? →'}
      </button>
    </div>
  );
}
