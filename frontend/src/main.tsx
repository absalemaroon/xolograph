import React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from '@/app/App';
import { Boot } from '@/app/Boot';
import { applySettings, useSettings } from '@/state/settingsStore';
import './styles/index.css';

applySettings(useSettings.getState());
useSettings.subscribe((s) => applySettings(s));

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <Boot>
      <App />
    </Boot>
  </React.StrictMode>,
);
