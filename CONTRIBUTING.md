# Contributing to Haws Volunteers

Thank you for your interest in contributing to the Haws Volunteers project! This document provides guidelines for contributing to the project.

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on what is best for the community
- Show empathy towards other community members

## How to Contribute

### Reporting Bugs

If you find a bug, please create an issue with:

1. A clear, descriptive title
2. Steps to reproduce the issue
3. Expected behavior
4. Actual behavior
5. Screenshots (if applicable)
6. Environment details (OS, Go version, Node version)

### Suggesting Enhancements

We welcome enhancement suggestions! Please create an issue with:

1. A clear description of the enhancement
2. Why this enhancement would be useful
3. Possible implementation approaches
4. Any potential drawbacks

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following our coding standards
3. **Test your changes** thoroughly
4. **Update documentation** if needed
5. **Write clear commit messages**
6. **Submit a pull request**

## Development Setup

See [SETUP.md](SETUP.md) for detailed setup instructions.

Quick start:
```bash
git clone https://github.com/networkengineer-cloud/go-volunteer-media.git
cd go-volunteer-media
make setup
make dev-backend  # In one terminal
make dev-frontend # In another terminal
```

## Coding Standards

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `go fmt` to format your code
- Write meaningful variable and function names
- Add comments for exported functions and complex logic
- Keep functions small and focused

Example:
```go
// GetUserByID retrieves a user from the database by their ID
func GetUserByID(db *gorm.DB, userID uint) (*models.User, error) {
    var user models.User
    if err := db.First(&user, userID).Error; err != nil {
        return nil, err
    }
    return &user, nil
}
```

### TypeScript/React Code

- Use TypeScript for type safety
- Follow React hooks best practices
- Use functional components
- Keep components small and reusable
- Use meaningful prop names

Example:
```typescript
interface AnimalCardProps {
  animal: Animal;
  onEdit: (id: number) => void;
  onDelete: (id: number) => void;
}

const AnimalCard: React.FC<AnimalCardProps> = ({ animal, onEdit, onDelete }) => {
  // Component implementation
};
```

### Commit Messages

Write clear, descriptive commit messages:

- Use present tense ("Add feature" not "Added feature")
- Start with a capital letter
- Keep the first line under 50 characters
- Add detailed description after a blank line if needed

Good examples:
```
Add animal image upload functionality

Implement image upload for animal profiles using multipart form data.
Includes validation for file size and type.
```

```
Fix authentication token expiration issue

Tokens were expiring too quickly. Changed expiration time from
1 hour to 24 hours and added refresh token logic.
```

## Project Structure

```
go-volunteer-media/
├── cmd/
│   └── api/              # Application entry point
├── internal/
│   ├── auth/             # Authentication logic
│   ├── database/         # Database connection and migrations
│   ├── handlers/         # HTTP request handlers
│   ├── middleware/       # HTTP middleware
│   └── models/           # Data models
├── frontend/
│   └── src/
│       ├── api/          # API client
│       ├── components/   # Reusable components
│       ├── contexts/     # React contexts
│       └── pages/        # Page components
├── API.md                # API documentation
├── DEPLOYMENT.md         # Deployment guide
├── SETUP.md              # Setup instructions
└── README.md             # Project overview
```

## Testing

### Backend Tests

Write tests for new functionality:

```go
func TestCreateAnimal(t *testing.T) {
    // Setup test database
    db := setupTestDB()
    defer teardownTestDB(db)

    // Create test data
    animal := models.Animal{
        Name:    "Test Dog",
        Species: "Dog",
        GroupID: 1,
    }

    // Test the function
    err := db.Create(&animal).Error
    assert.NoError(t, err)
    assert.NotZero(t, animal.ID)
}
```

Run tests:
```bash
go test ./...
```

### Frontend Tests

Consider adding tests for components and logic:

```typescript
import { render, screen } from '@testing-library/react';
import AnimalCard from './AnimalCard';

test('renders animal card with name', () => {
  const animal = { id: 1, name: 'Buddy', species: 'Dog' };
  render(<AnimalCard animal={animal} />);
  expect(screen.getByText('Buddy')).toBeInTheDocument();
});
```

## Documentation

- Update README.md for new features
- Update API.md for new endpoints
- Add inline comments for complex logic
- Update SETUP.md if setup process changes

## Areas for Contribution

Here are some areas where contributions would be especially valuable:

### Features
- [ ] User profile management
- [ ] Email notifications
- [ ] Image upload functionality
- [ ] Search and filter for animals
- [ ] Pagination for large datasets
- [ ] Mobile responsive improvements
- [ ] Dark mode support

### Testing
- [ ] Unit tests for handlers
- [ ] Integration tests for API
- [ ] Frontend component tests
- [ ] End-to-end tests

### Documentation
- [ ] Video tutorials
- [ ] More code examples
- [ ] Translation to other languages
- [ ] Architecture diagrams

### Infrastructure
- [ ] CI/CD pipeline
- [ ] Kubernetes deployment configs
- [ ] Monitoring and alerting
- [ ] Performance optimization

## Review Process

1. All pull requests require review before merging
2. At least one approval is needed
3. CI checks must pass
4. Code should follow project standards
5. Documentation should be updated if needed

## Getting Help

- Create an issue with the `question` label
- Join discussions in existing issues
- Check existing documentation

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (MIT License).

## Recognition

Contributors will be recognized in the project README and release notes.

Thank you for contributing to Haws Volunteers! 🐾
