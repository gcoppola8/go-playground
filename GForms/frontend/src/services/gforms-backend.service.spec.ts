import { TestBed } from '@angular/core/testing';

import { FormService } from './gforms-backend.service';

describe('GformsBackendService', () => {
  let service: FormService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(FormService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
