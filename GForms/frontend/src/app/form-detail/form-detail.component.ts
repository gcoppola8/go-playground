import { Component, OnInit, OnDestroy, CUSTOM_ELEMENTS_SCHEMA } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { FormService, Form, Question, FormResponse, NewFormResponse } from '../../services/gforms-backend.service'; // Adjust path if needed
import { Subscription } from 'rxjs';
import { CommonModule } from '@angular/common';
import { ClarityModule } from '@clr/angular'; // Import ClarityModule if using Clarity components

@Component({
  selector: 'app-form-detail',
  standalone: true,
  imports: [CommonModule, RouterLink, ClarityModule], 
  templateUrl: './form-detail.component.html',
  styleUrls: ['./form-detail.component.css'],
  schemas: [CUSTOM_ELEMENTS_SCHEMA]
})
export class FormDetailComponent implements OnInit, OnDestroy {
  form: Form | null = null;
  responses: FormResponse[] = []; // Placeholder for responses
  isLoading: boolean = true;
  error: string | null = null;
  private routeSub: Subscription | undefined;
  private formSub: Subscription | undefined;
  // private responsesSub: Subscription | undefined; // Placeholder for responses subscription

  constructor(
    private route: ActivatedRoute,
    private formService: FormService
  ) { }

  ngOnInit(): void {
    this.routeSub = this.route.paramMap.subscribe(params => {
      const formId = params.get('id');
      if (formId) {
        this.loadFormDetails(formId);
        this.loadFormResponses(formId); // Call this when response fetching is implemented
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
        // Only set isLoading to false if both form and responses (eventually) are loaded
        // For now, we set it here.
        this.isLoading = false;
        console.log('Form details loaded:', this.form);
      },
      error: (err) => {
        console.error('Error loading form details:', err);
        this.error = err.message || 'Could not load form details.';
        this.isLoading = false;
      }
    });
  }

  // Placeholder function to load responses - Implement this based on your API
  loadFormResponses(formId: string): void {
     console.log('Placeholder: Loading responses for form ID:', formId);
     this.isLoading = true; // Potentially manage loading state across both calls
     this.formService.getFormResponses(formId).subscribe({
      next: (data) => {
        this.responses = data;
        this.isLoading = false;
        console.log('Form responses loaded:', this.responses);
      },
      error: (err) => {
        console.error('Error loading form responses:', err);
        this.error = err.message || 'Could not load form responses.';
        this.isLoading = false;
      }
    });
     this.responses = []; // Reset or load actual data
  }

  retryLoad(): void {
    const formId = this.route.snapshot.paramMap.get('id');
    if (formId) {
       this.loadFormDetails(formId);
       // this.loadFormResponses(formId);
    }
  }

  sendResponse(): void {
    if (this.form?.questions === undefined || this.form?.questions.length === 0) {
      console.error('Form has no questions to submit.');
      return;
    }

    let nfr : NewFormResponse = {
      form_id: this.form!.id,
      respondent_user_id: 'user-id-1',
      answers: [
        {
          question_id: this.form!.questions[0].id,
          value: 'answer 1'
        },
        {
          question_id: this.form!.questions[1].id,
          value: 'answer 2'
        }
      ]
    };

    this.formService.submitFormResponse(this.form!.id, nfr).subscribe({
      next: (data) => {
        console.log('Form response submitted:', data);
      },
      error: (err) => {
        console.error('Error submitting form response:', err);
      }
    });
  }
}
