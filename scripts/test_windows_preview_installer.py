"""Guard the opt-in preview installer update policy; never alter public installers."""
import unittest

from package_windows_installer import manual_updates_definition


class PreviewUpdatePolicyTest(unittest.TestCase):
    def test_manual_mode_requires_private_preview(self):
        for channel in ('public-beta', 'unknown'):
            with self.subTest(channel=channel), self.assertRaisesRegex(ValueError, 'internal-preview'):
                manual_updates_definition(channel, True)
        self.assertIn('!define MANUAL_UPDATES', manual_updates_definition('internal-preview', True))

    def test_existing_installers_keep_their_update_policy(self):
        for channel in ('public-beta', 'internal-preview'):
            with self.subTest(channel=channel):
                self.assertEqual(manual_updates_definition(channel, False), '')


if __name__ == '__main__':
    unittest.main()
