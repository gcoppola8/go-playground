import { Component, OnInit, inject } from '@angular/core';
import { FormBuilder, FormGroup, Validators, ReactiveFormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { CommonModule } from '@angular/common';
import { FormService, NewFormDto } from '../../services/gforms-backend.service';
import { ClarityModule } from '@clr/angular';

@Component({
  selector: 'app-forms-new',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    ClarityModule
  ],
  templateUrl: './forms-new.component.html',
})
export class FormsNewComponent implements OnInit {

  private fb = inject(FormBuilder);
  private router = inject(Router);
  private formService = inject(FormService);

  newForm!: FormGroup;
  isSubmitting = false;
  errorMessage: string | null = null;

  ngOnInit(): void {
    this.newForm = this.fb.group({
      title: ['', [Validators.required, Validators.maxLength(100)]],
      description: ['', Validators.maxLength(500)]
    });
  }

  onSubmit(): void {
    if (this.newForm.invalid || this.isSubmitting) {
      this.newForm.markAllAsTouched();
      return;
    }

    this.isSubmitting = true;
    this.errorMessage = null;
    
    let formData : NewFormDto = {
      title: this.title?.value,
      description: this.description?.value,
      creator_user_id: '1',
      questions: []
    };

    console.log('Form data to submit:', formData);

    this.formService.createForm(formData).subscribe({
      next: (createdForm) => {
        console.log('Form created successfully:', createdForm);
        this.isSubmitting = false;
        alert('Form created successfully!');
        this.router.navigate(['/forms']);
      },
      error: (error) => {
        console.error('Error creating form:', error);
        this.errorMessage = 'An error occurred during creation. Please try again.';
        this.isSubmitting = false;
      }
    });

  }

  goBack(): void {
    this.router.navigate(['/forms']);
  }

  get title() {
    return this.newForm.get('title');
  }

  get description() {
    return this.newForm.get('description');
  }
}
