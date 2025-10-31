import { CommonModule } from '@angular/common';
import { Component } from '@angular/core';
import {
  FormBuilder,
  FormGroup,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { ClarityModule } from '@clr/angular';
import {
  AccountService,
  SignupDto,
} from '../../../services/account-service.service';

@Component({
  selector: 'app-registration',
  standalone: true,
  imports: [ReactiveFormsModule, CommonModule, ClarityModule],
  templateUrl: './registration.component.html',
  styleUrl: './registration.component.css',
})
export class RegistrationComponent {
  registrationForm: FormGroup;
  errorMessage: any;
  username: string = '';
  email: string = '';
  password: string = '';
  confirmPassword: string = '';
  passwordMatch: boolean = false;
  showSuccessModal: boolean = false;

  constructor(private fb: FormBuilder, private accountService: AccountService) {
    this.registrationForm = this.fb.group({
      username: ['', Validators.required],
      email: ['', [Validators.required, Validators.email]],
      password: ['', [Validators.required, Validators.minLength(6)]],
      confirmPassword: ['', Validators.required],
    });
  }

  onSubmit() {
    this.comparePasswords();

    if (this.registrationForm.valid && this.passwordMatch) {
      console.log('Form submitted', this.registrationForm.value);
      const signupDto: SignupDto = {
        username: this.registrationForm.value.username,
        email: this.registrationForm.value.email,
        password: this.registrationForm.value.password,
      };

      this.accountService.signup(signupDto).subscribe({
        next: (response) => {
          console.log('Registration successful', response);
          this.registrationForm.reset();
          this.showSuccessModal = true;
        },
        error: (error) => {
          console.error('Registration failed', error);
          this.errorMessage = error.message;
        },
      });
    } else {
      console.log('Form is invalid');
    }
  }

  onPasswordChange() {
    this.comparePasswords();
  }

  comparePasswords() {
    if (this.password !== this.confirmPassword) {
      this.errorMessage = 'Passwords do not match';
      this.passwordMatch = false;
    } else {
      this.errorMessage = '';
      this.passwordMatch = true;
    }
  }

  closeSuccessModal(): void {
    this.showSuccessModal = false;
  }
}
