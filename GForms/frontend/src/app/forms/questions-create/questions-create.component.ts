import { CommonModule } from '@angular/common';
import { Component, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  Form,
  FormService,
  Question,
} from '../../../services/gforms-backend.service';
import { Subscription } from 'rxjs';
import { ActivatedRoute, Router } from '@angular/router';
import { BrowserModule } from '@angular/platform-browser';
import { ClarityModule } from '@clr/angular';

class QuestionRequest {
  text!: string;
  type!: string;
  extra_info!: string;
  is_required!: boolean;
}

@Component({
  selector: 'app-questions-create',
  templateUrl: './questions-create.component.html',
  imports: [CommonModule, FormsModule, ClarityModule],
  standalone: true,
})
export class QuestionsCreateComponent implements OnInit {
  // Array to hold all the questions
  questions: Question[] = [];
  newQuestions: QuestionRequest[] = [];
  isLoading: boolean = false;
  error: string | null = null;
  forms: any;

  private routeSub: Subscription | undefined;
  private formSub: Subscription | undefined;
  form: Form | null = null;

  constructor(
    private formService: FormService,
    private route: ActivatedRoute,
    private router: Router
  ) { }

  // Initialize the component
  ngOnInit(): void {
    // Start with one empty question when the component loads
    this.addQuestion();

    this.routeSub = this.route.paramMap.subscribe((params) => {
      const formId = params.get('id');
      if (formId) {
        this.loadFormDetails(formId);
      } else {
        this.error = 'Form ID not found in route.';
        this.isLoading = false;
      }
    });
  }

  ngOnDestroy(): void {
    this.routeSub?.unsubscribe();
    this.formSub?.unsubscribe();
    // this.responsesSub?.unsubscribe(); // Unsubscribe from responses
  }

  loadFormDetails(id: string): void {
    this.isLoading = true;
    this.error = null;
    this.form = null;

    this.formSub = this.formService.getFormById(id).subscribe({
      next: (data) => {
        this.form = data;
        this.isLoading = false;
        console.log('Form details loaded:', this.form);
        if (this.form?.questions) {
          this.questions = this.form.questions;
        }
      },
      error: (err) => {
        console.error('Error loading form details:', err);
        this.error = err.message || 'Could not load form details.';
        this.isLoading = false;
      },
    });
  }

  // Adds a new blank question to the array
  addQuestion(): void {
    this.questions.push({
      id: crypto.randomUUID(),
      text: '',
      type: '', // Default to empty, user must select
      extra_info: '',
      is_required: false,
      created_at: '',
      updated_at: '',
      required: false,
      description: '',
      placeholder: ''
    });
  }

  // Removes a question from the array based on its index
  removeQuestion(index: number): void {
    // Prevent removing the last question if you always want at least one
    // if (this.questions.length > 1) {
    this.questions.splice(index, 1);
    // } else {
    //   console.warn("Cannot remove the last question.");
    //   // Optionally, provide user feedback (e.g., using a toast notification)
    // }
  }

  // Handles the form submission
  onSubmit(): void {
    console.log('Form Submitted!');
    console.log('Questions Data:', this.questions);
    if (this.isFormValid()) {
      console.log('Form is valid, proceeding with submission...');

      this.questions.map((q) => {
        const questionRequest: QuestionRequest = {
          text: q.text,
          type: q.type,
          extra_info: q.extra_info,
          is_required: q.is_required,
        };
      });

      this.formService.updateQuestions(this.form?.id || '', this.questions).subscribe({
        next: (response) => {
          console.log('Form updated successfully:', response);
          alert('Form updated successfully!');
          // redirect to the form details
          this.router.navigate(['/forms', this.form?.id]);
        },
        error: (err) => {
          console.error('Error updating form:', err);
          this.error = err.message || 'Could not update form.';
        }
      });
    } else {
      console.error('Form is invalid. Please check the fields.');
      // Optionally, provide user feedback
    }
  }

  // Optional: Basic validation check
  isFormValid(): boolean {
    return this.questions.every((q) => q.text && q.type);
    // Add more complex validation as needed
  }

  clearOptions(idx: number) {
    this.questions[idx].extra_info = '';
  }
}
