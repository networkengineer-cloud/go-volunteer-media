import React from 'react';
import { Link } from 'react-router-dom';
import './Home.css';

const Home: React.FC = () => {
  return (
    <main className="home">
      <section className="hero">
        <div className="hero-overlay" />
        <div className="hero-content">
          <h1>Help Pets. Help People.</h1>
          <p>Join our community of volunteers to support animals and the humans who love them.</p>
          <div className="hero-cta">
            <Link to="/login" className="btn btn-primary">Login to Continue</Link>
          </div>
        </div>
      </section>

      <section className="features">
        <div className="feature">
          <h3>Foster & Adopt</h3>
          <p>Share updates and media for animals looking for forever homes.</p>
        </div>
        <div className="feature">
          <h3>Volunteer Groups</h3>
          <p>Collaborate with focused teams like Dogs, Cats, and Mod Squad.</p>
        </div>
        <div className="feature">
          <h3>Community Updates</h3>
          <p>Post stories and progress updates to engage your community.</p>
        </div>
      </section>
    </main>
  );
};

export default Home;
