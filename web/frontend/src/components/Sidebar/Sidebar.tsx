import { useEffect, useState } from 'react';
import { useSelector } from 'react-redux';
import { useNavigate, useLocation } from 'react-router-dom';
import {
  Drawer, List, ListItemButton, ListItemIcon, ListItemText, Typography, Box, Chip,
  Avatar, Menu, MenuItem, Divider, Skeleton,
} from '@mui/material';
import {
  Dashboard, TrendingUp, Cloud, AccountBalance, Lightbulb, Extension, Settings,
  FolderSpecial, Person, Logout,
} from '@mui/icons-material';
import type { RootState } from '../../redux/store';
import { api, type UserInfo } from '../../lib/api';

const iconMap: Record<string, React.ReactElement> = {
  Dashboard: <Dashboard />,
  FolderSpecial: <FolderSpecial />,
  TrendingUp: <TrendingUp />,
  Cloud: <Cloud />,
  AccountBalance: <AccountBalance />,
  Lightbulb: <Lightbulb />,
  Extension: <Extension />,
  Settings: <Settings />,
};

const DRAWER_WIDTH = 240;

function getInitials(name: string): string {
  return name
    .split(/\s+/)
    .slice(0, 2)
    .map(w => w[0]?.toUpperCase() ?? '')
    .join('');
}

export default function Sidebar() {
  const entries = useSelector((state: RootState) => state.sidebar.entries);
  const navigate = useNavigate();
  const location = useLocation();
  const theme = useSelector((state: RootState) => {
    const name = state.theme.name;
    return state.theme.appThemes.find(t => t.name === name) ?? state.theme.appThemes[0];
  });

  const [user, setUser] = useState<UserInfo | null>(null);
  const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);

  useEffect(() => {
    api.get<UserInfo>('/me').then(setUser).catch(() => {});
  }, []);

  const handleMenuClose = () => setMenuAnchor(null);

  return (
    <Drawer
      variant="permanent"
      sx={{
        width: DRAWER_WIDTH,
        flexShrink: 0,
        '& .MuiDrawer-paper': {
          width: DRAWER_WIDTH,
          boxSizing: 'border-box',
          bgcolor: theme?.sidebar?.background || 'background.paper',
          borderRight: '1px solid',
          borderColor: 'divider',
          display: 'flex',
          flexDirection: 'column',
        },
      }}
    >
      <Box sx={{ p: 2, pb: 1 }}>
        <Typography variant="h6" fontWeight={700} color="primary">
          FinGuard
        </Typography>
        <Chip label="FinOps" size="small" color="primary" variant="outlined" sx={{ mt: 0.5 }} />
      </Box>
      <List sx={{ flex: 1, pt: 1 }}>
        {entries.map(entry => {
          const isActive = entry.url === '/'
            ? location.pathname === '/'
            : location.pathname.startsWith(entry.url);
          return (
            <ListItemButton
              key={entry.name}
              selected={isActive}
              onClick={() => navigate(entry.url)}
              sx={{
                mx: 1,
                borderRadius: 1,
                mb: 0.5,
                '&.Mui-selected': {
                  bgcolor: theme?.sidebar?.selectedBackground || 'action.selected',
                  color: theme?.sidebar?.selectedColor || 'primary.main',
                  '& .MuiListItemIcon-root': {
                    color: theme?.sidebar?.selectedColor || 'primary.main',
                  },
                },
              }}
            >
              <ListItemIcon sx={{ minWidth: 36, color: theme?.sidebar?.color || 'text.secondary' }}>
                {iconMap[entry.icon || ''] || <Dashboard />}
              </ListItemIcon>
              <ListItemText
                primary={entry.label}
                primaryTypographyProps={{ fontSize: '0.875rem' }}
              />
            </ListItemButton>
          );
        })}
      </List>

      <Divider />
      <Box
        sx={{
          p: 1.5,
          display: 'flex',
          alignItems: 'center',
          gap: 1.5,
          cursor: 'pointer',
          '&:hover': { bgcolor: 'action.hover' },
          borderRadius: 1,
          m: 1,
        }}
        onClick={e => setMenuAnchor(e.currentTarget)}
      >
        {user ? (
          <>
            <Avatar
              sx={{
                width: 32,
                height: 32,
                fontSize: '0.8rem',
                bgcolor: 'primary.main',
                color: 'primary.contrastText',
              }}
            >
              {getInitials(user.displayName || user.email)}
            </Avatar>
            <Box sx={{ overflow: 'hidden', flex: 1 }}>
              <Typography variant="body2" fontWeight={600} noWrap>
                {user.displayName || user.email}
              </Typography>
              <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block' }}>
                {user.email}
              </Typography>
            </Box>
          </>
        ) : (
          <>
            <Skeleton variant="circular" width={32} height={32} />
            <Box sx={{ flex: 1 }}>
              <Skeleton width="80%" height={16} />
              <Skeleton width="60%" height={14} />
            </Box>
          </>
        )}
      </Box>

      <Menu
        anchorEl={menuAnchor}
        open={Boolean(menuAnchor)}
        onClose={handleMenuClose}
        anchorOrigin={{ vertical: 'top', horizontal: 'center' }}
        transformOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <MenuItem
          onClick={() => {
            handleMenuClose();
            navigate('/account');
          }}
        >
          <ListItemIcon><Person fontSize="small" /></ListItemIcon>
          <ListItemText>My Account</ListItemText>
        </MenuItem>
        <Divider />
        <MenuItem
          onClick={() => {
            handleMenuClose();
            window.location.href = '/logout';
          }}
        >
          <ListItemIcon><Logout fontSize="small" /></ListItemIcon>
          <ListItemText>Log Out</ListItemText>
        </MenuItem>
      </Menu>
    </Drawer>
  );
}
