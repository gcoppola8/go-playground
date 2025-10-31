import { CommonModule } from '@angular/common';
import { Component } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';
import { ClarityModule } from '@clr/angular';
import { AccountService } from '../../../services/account-service.service';
import { Router } from '@angular/router';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [ReactiveFormsModule, CommonModule, ClarityModule],
  templateUrl: './login.component.html',
  styleUrl: './login.component.css',
})
export class LoginComponent {
  loginForm: FormGroup;
  errorMessage: string | null = null; 
  showSuccessModal: boolean = false;


  constructor(private fb: FormBuilder, private accountService: AccountService, private router: Router) {
    this.loginForm = this.fb.group({
      email: ['', [Validators.required, Validators.email]],
      password: ['', [Validators.required, Validators.minLength(6)]],
    });
  }

  onSubmit() {
    if (this.loginForm.valid) {
      const signinDto = {
        email: this.loginForm.value.email,
        password: this.loginForm.value.password,
      };
      this.accountService.signin(signinDto).subscribe({
        next: (response) => {
          console.log('Login successful', response);
          this.loginForm.reset();
          this.showSuccessModal = true;
          this.router.navigate(['/forms']);
        },
        error: (error) => {
          console.error('Login failed', error);
          this.errorMessage = error.message; 
        },
      });
    } else {
      console.log('Form is invalid');
    }
  }

  closeSuccessModal(): void {
    this.showSuccessModal = false;
  }
}
