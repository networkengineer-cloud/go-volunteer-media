import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { settingsApi } from '../api/client';
import './Home.css';

const Home: React.FC = () => {
  const [heroImage, setHeroImage] = useState<string>('https://images.unsplash.com/photo-1518806118471-f28b20a1d79d?q=80&w=1920&auto=format&fit=crop');

  useEffect(() => {
    const loadSettings = async () => {
      try {
        const response = await settingsApi.getAll();
        console.log('Settings response:', response.data);
        if (response.data && response.data.hero_image_url) {
          console.log('Setting hero image to:', response.data.hero_image_url);
          setHeroImage(response.data.hero_image_url);
        } else {
          console.log('No hero_image_url in response, using default');
        }
      } catch (error) {
        console.error('Failed to load site settings:', error);
        // Keep default image on error
      }
    };
    loadSettings();
  }, []);

  console.log('Current heroImage:', heroImage);

  const heroStyle: React.CSSProperties = {
    backgroundImage: `url(${heroImage})`,
    backgroundSize: 'cover',
    backgroundPosition: 'center',
    backgroundRepeat: 'no-repeat'
  };

  return (
    <main className="home">
      <section className="hero" style={heroStyle}>
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
