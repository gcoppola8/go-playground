import { Routes } from '@angular/router';
import { FormDetailComponent } from './form-detail/form-detail.component';
import { FormsListComponent } from './forms-list/forms-list.component';
import { FormsNewComponent } from './forms-new/forms-new.component';
import { ResponsesNewComponent } from './responses-new/responses-new.component';
import { LoginComponent } from './account/login/login.component';
import { RegistrationComponent } from './account/registration/registration.component';
import { QuestionsCreateComponent } from './forms/questions-create/questions-create.component';

export const routes: Routes = [
    { path: 'forms', component: FormsListComponent },
    { path: 'forms/new', component: FormsNewComponent },
    { path: 'forms/:id', component: FormDetailComponent },
    { path: 'forms/:id/respond', component: ResponsesNewComponent},
    { path: 'forms/:id/questions', component: QuestionsCreateComponent},

    // Account routes
    { path: 'account/login', component: LoginComponent},
    { path: 'account/register', component: RegistrationComponent},

    { path: '', redirectTo: '/forms', pathMatch: 'full'}
];
