import { Component, inject, type OnDestroy, type OnInit } from '@angular/core'
import { MaterialModule } from '../../material-module'
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms'
import { ValidationErrorPipe } from '../../pipes/ValidationErrorPipe'
import { SnackbarService } from '../../services/snackbar.service'
import { UserService } from '../../services/user.service'
import type { CurrentUserDetails } from '@shared/api-response/UserDetails'
import { ConfigService } from '../../services/config.service'
import { PasswordSetComponent } from '../../components/password-reset/password-set.component'
import { SpinnerService } from '../../services/spinner.service'
import { PasskeyService, type PasskeySupport } from '../../services/passkey.service'
import { WebAuthnAbortService, WebAuthnError } from '@simplewebauthn/browser'
import type { ConfigResponse } from '@shared/api-response/ConfigResponse'
import { MatDialog } from '@angular/material/dialog'
import { ConfirmComponent } from '../../dialogs/confirm/confirm.component'
import { TotpRegisterComponent } from '../../dialogs/totp-register/totp-register.component'
import { TranslateModule } from '@ngx-translate/core'
import { I18nService } from '../../services/i18n.service'

@Component({
  selector: 'app-home',
  imports: [
    ReactiveFormsModule,
    MaterialModule,
    ValidationErrorPipe,
    PasswordSetComponent,
    TranslateModule,
  ],
  templateUrl: './home.component.html',
  styleUrl: './home.component.scss',
})
export class HomeComponent implements OnInit, OnDestroy {
  user?: CurrentUserDetails
  public passkeySupport?: PasskeySupport
  public isPasskeySession: boolean = false
  config?: ConfigResponse

  public profileForm = new FormGroup({
    name: new FormControl<string>({
      value: '',
      disabled: false,
    }, [Validators.minLength(3)]),
  })

  public emailForm = new FormGroup({
    email: new FormControl<string>({
      value: '',
      disabled: false,
    }, [Validators.required, Validators.email]),
  })

  public passwordForm = new FormGroup({
    oldPassword: new FormControl<string>({
      value: '',
      disabled: false,
    }, []),
    newPassword: new FormControl<string>({
      value: '',
      disabled: false,
    }, [Validators.required]),
    confirmPassword: new FormControl<string>({
      value: '',
      disabled: false,
    }, [Validators.required]),
  }, {
    validators: (g) => {
      const passAreEqual = g.get('newPassword')?.value === g.get('confirmPassword')?.value
      if (!passAreEqual) {
        g.get('confirmPassword')?.setErrors({ notEqual: 'Must equal Password' })
        return { notEqual: 'Passwords do not match' }
      }
      g.get('confirmPassword')?.setErrors(null)
      return null
    },
  })

  private configService = inject(ConfigService)
  private userService = inject(UserService)
  private snackbarService = inject(SnackbarService)
  private spinnerService = inject(SpinnerService)
  passkeyService = inject(PasskeyService)
  private dialog = inject(MatDialog)
  private i18nService = inject(I18nService)

  async ngOnInit() {
    await this.loadUser()

    this.passkeySupport = await this.passkeyService.getPasskeySupport()
    this.config = await this.configService.getConfig()
  }

  ngOnDestroy(): void {
    WebAuthnAbortService.cancelCeremony()
  }

  async loadUser() {
    try {
      this.spinnerService.show()

      try {
        this.user = await this.userService.getMyUser({
          disableCache: true,
        })
      } catch (_e) {
        // If user cannot be loaded, refresh page
        location.reload()
        return
      }

      this.isPasskeySession = this.userService.passkeySession(this.user)

      this.profileForm.reset({
        name: this.user.name ?? '',
      })
      this.emailForm.reset({
        email: this.user.email,
      })
      this.passwordForm.reset()

      if (this.user.hasPassword) {
        this.passwordForm.controls.oldPassword.addValidators(Validators.required)
        this.passwordForm.controls.oldPassword.updateValueAndValidity()
      }
    } finally {
      this.spinnerService.hide()
    }
  }

  async updateProfile() {
    try {
      this.spinnerService.show()

      await this.userService.updateProfile({
        name: this.profileForm.value.name ?? undefined,
      })
      this.snackbarService.message(this.i18nService.instant('error.profileUpdated'))
    } catch (_e) {
      this.snackbarService.error(this.i18nService.instant('error.couldNotUpdateProfile'))
    } finally {
      await this.loadUser()
      this.spinnerService.hide()
    }
  }

  async updatePassword() {
    try {
      this.spinnerService.show()
      const { oldPassword, newPassword } = this.passwordForm.getRawValue()
      if (!newPassword) {
        throw new Error('Password missing.')
      }

      await this.userService.updatePassword({
        oldPassword: oldPassword,
        newPassword: newPassword,
      })
      this.snackbarService.message(this.i18nService.instant('error.passwordUpdated'))
      await this.loadUser()
    } catch (_e) {
      this.snackbarService.error(this.i18nService.instant('error.couldNotUpdatePassword'))
    } finally {
      this.spinnerService.hide()
    }
  }

  async updateEmail() {
    try {
      this.spinnerService.show()
      const email = this.emailForm.value.email
      if (!email) {
        throw new Error('Email missing.')
      }
      await this.userService.updateEmail({
        email: email,
      })
      // if email verification enabled, indicate that in message
      if (this.config?.emailVerification) {
        this.snackbarService.message(this.i18nService.instant('error.verificationEmailSent'))
      } else {
        this.snackbarService.message(this.i18nService.instant('error.emailUpdated'))
      }
    } catch (e) {
      console.error(e)
      this.snackbarService.error(this.i18nService.instant('error.couldNotUpdateEmail'))
    } finally {
      await this.loadUser()
      this.spinnerService.hide()
    }
  }

  async registerPasskey() {
    this.spinnerService.show()
    try {
      await this.passkeyService.register()
      await this.loadUser()
      this.snackbarService.message(this.i18nService.instant('error.passkeyRegisteredSuccess'))
    } catch (error) {
      if (error instanceof WebAuthnError && error.name === 'InvalidStateError') {
        this.snackbarService.error(this.i18nService.instant('error.passkeyAlreadyRegistered'))
      } else {
        this.snackbarService.error(this.i18nService.instant('error.couldNotRegisterPasskey'))
      }
      console.error(error)
    } finally {
      this.spinnerService.hide()
    }
  }

  addAuthenticator() {
    const hadTotp = this.user?.hasTotp
    const dialogRef = this.dialog.open<TotpRegisterComponent, { enableMfa: boolean } | undefined>(TotpRegisterComponent, {
      data: { enableMfa: true },
      panelClass: 'overflow-auto',
    })

    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        await this.loadUser()
        this.snackbarService.message(hadTotp ? this.i18nService.instant('error.authenticatorAdded') : this.i18nService.instant('error.mfaEnabled'))
      }
    })
  }

  removeAllPasskeys() {
    const dialogRef = this.dialog.open(ConfirmComponent, {
      data: {
        message: this.i18nService.instant('common.removeAllPasskeys') + '. ' + this.i18nService.instant('common.mfaNotEnabled'),
        header: this.i18nService.instant('common.delete'),
      },
    })

    dialogRef.afterClosed().subscribe(async (result) => {
      if (!result) {
        return
      }

      try {
        this.spinnerService.show()
        await this.userService.removeAllPasskeys()
        this.passkeyService.resetPasskeySeen()
        this.passkeyService.resetPasskeySkipped()
        this.snackbarService.message(this.i18nService.instant('error.removedAllPasskeys'))
      } catch (_e) {
        this.snackbarService.error(this.i18nService.instant('error.couldNotRemoveAllPasskeys'))
      } finally {
        await this.loadUser()
        this.spinnerService.hide()
      }
    })
  }

  removePassword() {
    const dialogRef = this.dialog.open(ConfirmComponent, {
      data: {
        message: this.i18nService.instant('common.removePassword') + '. ' + this.i18nService.instant('common.logoutTitle'),
        header: this.i18nService.instant('common.delete'),
      },
    })

    dialogRef.afterClosed().subscribe(async (result) => {
      if (!result) {
        return
      }

      try {
        this.spinnerService.show()
        await this.userService.removePassword()
        this.snackbarService.message(this.i18nService.instant('error.removedPassword'))
      } catch (_e) {
        this.snackbarService.error(this.i18nService.instant('error.couldNotRemovePassword'))
      } finally {
        await this.loadUser()
        this.spinnerService.hide()
      }
    })
  }

  removeAllAuthenticators() {
    const dialogRef = this.dialog.open(ConfirmComponent, {
      data: {
        message: this.i18nService.instant('common.removeAuthenticators'),
        header: this.i18nService.instant('common.delete'),
      },
    })

    dialogRef.afterClosed().subscribe(async (result) => {
      if (!result) {
        return
      }

      try {
        this.spinnerService.show()
        await this.userService.removeAllAuthenticators()
        this.snackbarService.message(this.i18nService.instant('error.mfaDisabled'))
      } catch (_e) {
        this.snackbarService.error(this.i18nService.instant('error.couldNotDisableMfa'))
      } finally {
        await this.loadUser()
        this.spinnerService.hide()
      }
    })
  }

  deleteUser() {
    const dialogRef = this.dialog.open(ConfirmComponent, {
      data: {
        message: this.i18nService.instant('common.deleteAccount') + '.',
        header: 'DANGER',
        requiredText: this.user?.username,
      },
    })

    dialogRef.afterClosed().subscribe(async (result) => {
      if (!result) {
        return
      }

      try {
        this.spinnerService.show()
        await this.userService.deleteUser()
        this.snackbarService.message(this.i18nService.instant('error.deletedAccount'))
      } catch (_e) {
        this.snackbarService.error(this.i18nService.instant('error.couldNotDeleteAccount'))
      } finally {
        await this.loadUser()
        this.spinnerService.hide()
      }
    })
  }
}
