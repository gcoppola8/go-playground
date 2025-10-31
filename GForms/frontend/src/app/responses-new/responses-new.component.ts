import {
  Component,
  OnInit,
  OnDestroy,
  CUSTOM_ELEMENTS_SCHEMA,
} from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import {
  FormService,
  Form,
  Question,
  FormResponse,
  NewFormResponse,
} from '../../services/gforms-backend.service'; // Adjust path if needed
import { Subscription } from 'rxjs';
import { CommonModule } from '@angular/common';
import { ClarityModule } from '@clr/angular'; // Import ClarityModule if using Clarity components
import {
  FormControl,
  FormGroup,
  FormsModule,
  ReactiveFormsModule,
  FormArray,
  FormBuilder,
} from '@angular/forms';

@Component({
  selector: 'app-responses-new',
  imports: [CommonModule, ClarityModule, ReactiveFormsModule, RouterLink],
  standalone: true,
  providers: [FormService],
  templateUrl: './responses-new.component.html',
})
export class ResponsesNewComponent implements OnInit, OnDestroy {
  form: Form | null = null;
  formId: string | null = '';
  isLoading: boolean = true;
  error: string | null = null;

  private routeSub: Subscription | undefined;
  private formSub: Subscription | undefined;
  QuestionsForm: FormGroup = new FormGroup({});

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private formService: FormService,
    private fb: FormBuilder
  ) {}

  ngOnInit(): void {
    this.routeSub = this.route.paramMap.subscribe((params) => {
      this.formId = params.get('id');
      if (this.formId) {
        this.loadFormDetails(this.formId);
      } else {
        this.error = 'Form ID not found in route.';
        this.isLoading = false;
      }
    });
  }

  ngOnDestroy(): void {
    this.routeSub?.unsubscribe();
    this.formSub?.unsubscribe();
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
          this.buildFormControls();
        }
      },
      error: (err) => {
        console.error('Error loading form details:', err);
        this.error = err.message || 'Could not load form details.';
        this.isLoading = false;
      },
    });
  }

  private buildFormControls(): void {
    if (!this.form?.questions) return;

    // Reset the form
    this.QuestionsForm = new FormGroup({});

    this.form.questions.forEach((question) => {
      if (question.type === 'checkbox') {
        // For checkboxes, create a FormGroup with a control for each option
        const options = (question.extra_info || '')
          .split(',')
          .map((opt) => opt.trim())
          .filter(opt => opt.length > 0);
        
        const group: { [key: string]: FormControl } = {};
        options.forEach((option) => {
          group[option] = new FormControl(false);
        });
        this.QuestionsForm.addControl(question.id, new FormGroup(group));
      } else {
        // For other types, create a single FormControl
        let defaultValue = '';
        if (question.type === 'number') {
          defaultValue = String(0);
        }
        this.QuestionsForm.addControl(question.id, new FormControl(defaultValue));
      }
    });
  }

  sendResponse(): void {
    if (
      this.form?.questions === undefined ||
      this.form?.questions.length === 0
    ) {
      console.log('Form has no questions to submit.');
      return;
    }

    let nfr: NewFormResponse = {
      form_id: this.form!.id,
      respondent_user_id:
        'user-id-' + Math.floor(Math.random() * 1000000000000000),
      answers: [],
    };

    this.form!.questions.forEach((question) => {
      const control = this.QuestionsForm.get(question.id);
      let value = '';

      if (question.type === 'checkbox') {
        // For checkboxes, collect all selected options
        const checkboxGroup = control as FormGroup;
        const selectedOptions: string[] = [];
        
        Object.keys(checkboxGroup.controls).forEach(option => {
          if (checkboxGroup.get(option)?.value === true) {
            selectedOptions.push(option);
          }
        });
        
        value = selectedOptions.join(', ');
      } else {
        // For other types, get the direct value
        value = String(control?.value || '');
      }

      nfr.answers?.push({
        question_id: question.id,
        value: value,
      });
    });

    console.log('Submitting form response:', nfr);

    this.formService.submitFormResponse(this.form!.id, nfr).subscribe({
      next: (data) => {
        alert('Form response submitted successfully!');
        console.log('Form response submitted:', data);
        this.router.navigate(['/forms', this.form!.id]);
      },
      error: (err) => {
        console.error('Error submitting form response:', err);
        alert('Error submitting form response. Please try again.');
      },
    });
  }

  // Helper method to get checkbox options for template
  getCheckboxOptions(question: Question): string[] {
    if (question.type !== 'checkbox') return [];
    return (question.extra_info || '')
      .split(',')
      .map((opt) => opt.trim())
      .filter(opt => opt.length > 0);
  }

  // Helper method to check if a checkbox option is selected
  isCheckboxOptionSelected(questionId: string, option: string): boolean {
    const control = this.QuestionsForm.get(questionId) as FormGroup;
    return control?.get(option)?.value === true;
  }

  // Helper method to toggle checkbox option
  toggleCheckboxOption(questionId: string, option: string): void {
    const control = this.QuestionsForm.get(questionId) as FormGroup;
    const optionControl = control?.get(option);
    if (optionControl) {
      optionControl.setValue(!optionControl.value);
    }
  }
}