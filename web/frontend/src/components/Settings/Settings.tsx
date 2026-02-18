import { useSelector, useDispatch } from 'react-redux';
import {
  Box, Typography, Card, CardContent, FormControl, InputLabel, Select, MenuItem,
} from '@mui/material';
import Grid from '@mui/material/Grid2';
import type { RootState } from '../../redux/store';
import { setTheme } from '../../redux/themeSlice';

export default function SettingsPage() {
  const dispatch = useDispatch();
  const themeName = useSelector((state: RootState) => state.theme.name);
  const themes = useSelector((state: RootState) => state.theme.appThemes);

  return (
    <Box>
      <Typography variant="h5" fontWeight={600} mb={3}>Settings</Typography>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" mb={2}>Appearance</Typography>
              <FormControl fullWidth>
                <InputLabel>Theme</InputLabel>
                <Select value={themeName} label="Theme"
                  onChange={e => dispatch(setTheme(e.target.value))}>
                  {themes.map(t => (
                    <MenuItem key={t.name} value={t.name}>
                      {t.name} ({t.base})
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}
