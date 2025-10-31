import { bootstrapApplication } from '@angular/platform-browser';
import { appConfig } from './app/app.config';
import { AppComponent } from './app/app.component';
import { ClarityIcons, cogIcon } from '@cds/core/icon';

// Register the icon(s)
ClarityIcons.addIcons(cogIcon);

bootstrapApplication(AppComponent, appConfig)
  .catch((err) => console.error(err));
